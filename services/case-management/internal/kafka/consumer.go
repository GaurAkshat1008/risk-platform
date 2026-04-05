package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"case-management/internal/cache"
	"case-management/internal/db"
	"case-management/internal/telemetry"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// DecisionMadeEvent matches the schema published by the Decision Service.
type DecisionMadeEvent struct {
	EventType       string   `json:"event_type"`
	DecisionID      string   `json:"decision_id"`
	PaymentEventID  string   `json:"payment_event_id"`
	TenantID        string   `json:"tenant_id"`
	Outcome         string   `json:"outcome"`
	ReasonCodes     []string `json:"reason_codes"`
	ConfidenceScore float64  `json:"confidence_score"`
	LatencyMs       int64    `json:"latency_ms"`
	Overridden      bool     `json:"overridden"`
	OccurredAt      string   `json:"occurred_at"`
}

// ConsumerConfig holds configuration for the Kafka consumer.
type ConsumerConfig struct {
	Brokers       string
	DecisionTopic string
	ConsumerGroup string
}

// Consumer reads decision.made events and creates cases for review/block decisions.
type Consumer struct {
	reader    *kafka.Reader
	store     *db.CaseStore
	publisher *CaseEventPublisher
	slaCache  *cache.SLACache
	metrics   *telemetry.Metrics
	logger    *slog.Logger
}

func NewConsumer(
	cfg ConsumerConfig,
	store *db.CaseStore,
	publisher *CaseEventPublisher,
	slaCache *cache.SLACache,
	metrics *telemetry.Metrics,
	logger *slog.Logger,
) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        strings.Split(cfg.Brokers, ","),
		Topic:          cfg.DecisionTopic,
		GroupID:        cfg.ConsumerGroup,
		MinBytes:       1,
		MaxBytes:       10 << 20,
		CommitInterval: 0,
	})
	return &Consumer{
		reader:    r,
		store:     store,
		publisher: publisher,
		slaCache:  slaCache,
		metrics:   metrics,
		logger:    logger,
	}
}

// Run blocks, consuming decision.made messages until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) {
	c.logger.Info("case-management kafka consumer started", "topic", c.reader.Config().Topic)
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.logger.Info("case consumer stopping")
				return
			}
			c.logger.Error("kafka read error", "error", err)
			continue
		}

		c.metrics.KafkaConsumeTotal.Add(ctx, 1,
			metric.WithAttributes(attribute.String("topic", msg.Topic)))

		var event DecisionMadeEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.Error("unmarshal decision.made failed",
				"partition", msg.Partition, "offset", msg.Offset, "error", err)
			continue
		}

		// Only create cases for review and block outcomes
		if event.Outcome != "review" && event.Outcome != "block" {
			continue
		}

		priority := outcomeToPriority(event.Outcome)
		slaDeadline := priorityToSLADeadline(priority)

		caseRecord, err := c.store.CreateCase(
			ctx,
			event.DecisionID,
			event.TenantID,
			event.PaymentEventID,
			event.Outcome,
			priority,
			slaDeadline,
		)
		if err != nil {
			c.logger.Error("create case failed",
				"decision_id", event.DecisionID,
				"tenant_id", event.TenantID,
				"error", err)
			continue
		}

		c.metrics.CasesCreatedTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("outcome", event.Outcome),
				attribute.String("priority", priority),
			))

		// Register SLA in Redis cache
		if err := c.slaCache.Register(ctx, caseRecord.ID, slaDeadline); err != nil {
			c.logger.Warn("failed to register SLA in cache", "case_id", caseRecord.ID, "error", err)
		}

		// Publish case.created event
		if err := c.publisher.PublishCaseCreated(ctx, caseRecord); err != nil {
			c.logger.Warn("publish case.created failed (best-effort)", "error", err)
		} else {
			c.metrics.KafkaPublishTotal.Add(ctx, 1,
				metric.WithAttributes(attribute.String("topic", "case.created")))
		}

		c.logger.Info("case created",
			"case_id", caseRecord.ID,
			"decision_id", event.DecisionID,
			"tenant_id", event.TenantID,
			"priority", priority,
			"sla_deadline", slaDeadline.Format(time.RFC3339),
		)
	}
}

func (c *Consumer) Close() {
	if err := c.reader.Close(); err != nil {
		c.logger.Error("kafka reader close error", "error", err)
	}
}

// outcomeToPriority maps a decision outcome to a case priority.
func outcomeToPriority(outcome string) string {
	switch outcome {
	case "block":
		return "critical"
	case "review":
		return "high"
	default:
		return "medium"
	}
}

// priorityToSLADeadline returns the SLA deadline for a given priority level.
func priorityToSLADeadline(priority string) time.Time {
	now := time.Now().UTC()
	switch priority {
	case "critical":
		return now.Add(4 * time.Hour)
	case "high":
		return now.Add(24 * time.Hour)
	case "medium":
		return now.Add(48 * time.Hour)
	default:
		return now.Add(72 * time.Hour)
	}
}
