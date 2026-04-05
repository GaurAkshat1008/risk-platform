package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"decision/internal/db"
	"decision/internal/telemetry"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// RiskEvaluatedEvent matches the schema published by the Risk Orchestrator.
type RiskEvaluatedEvent struct {
	EventType       string              `json:"event_type"`
	PaymentEventID  string              `json:"payment_event_id"`
	TenantID        string              `json:"tenant_id"`
	AggregateAction string              `json:"aggregate_action"`
	FailOpen        bool                `json:"fail_open"`
	RuleResults     []RuleResultSummary `json:"rule_results"`
	LatencyMs       int64               `json:"latency_ms"`
	EvaluatedAt     time.Time           `json:"evaluated_at"`
}

type RuleResultSummary struct {
	RuleID   string `json:"rule_id"`
	RuleName string `json:"rule_name"`
	Matched  bool   `json:"matched"`
	Action   string `json:"action"`
	Reason   string `json:"reason"`
}

type ConsumerConfig struct {
	Brokers       string
	RiskTopic     string
	ConsumerGroup string
}

// Consumer reads risk.evaluated events from Kafka and records decisions.
type Consumer struct {
	reader    *kafka.Reader
	store     *db.DecisionStore
	publisher *DecisionEventPublisher
	metrics   *telemetry.Metrics
	logger    *slog.Logger
}

func NewConsumer(
	cfg ConsumerConfig,
	store *db.DecisionStore,
	publisher *DecisionEventPublisher,
	metrics *telemetry.Metrics,
	logger *slog.Logger,
) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        strings.Split(cfg.Brokers, ","),
		Topic:          cfg.RiskTopic,
		GroupID:        cfg.ConsumerGroup,
		MinBytes:       1,
		MaxBytes:       10 << 20,
		CommitInterval: 0,
	})
	return &Consumer{
		reader:    r,
		store:     store,
		publisher: publisher,
		metrics:   metrics,
		logger:    logger,
	}
}

// Run blocks, consuming risk.evaluated messages until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) {
	c.logger.Info("decision kafka consumer started", "topic", c.reader.Config().Topic)
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.logger.Info("decision consumer stopping")
				return
			}
			c.logger.Error("kafka read error", "error", err)
			continue
		}

		start := time.Now()
		c.metrics.KafkaConsumeTotal.Add(ctx, 1)

		var event RiskEvaluatedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.Error("unmarshal risk.evaluated failed",
				"partition", msg.Partition, "offset", msg.Offset, "error", err)
			continue
		}

		spanCtx, span := telemetry.SpanFromContext(ctx, "decision.consumer", "ProcessRiskEvaluated")

		log := c.logger.With(
			"payment_event_id", event.PaymentEventID,
			"tenant_id", event.TenantID,
			"aggregate_action", event.AggregateAction,
		)

		// Build reason codes from rule results
		reasonCodes := buildReasonCodes(event)

		// Marshal rule results for storage
		ruleResultsJSON, _ := json.Marshal(event.RuleResults)

		decision, err := c.store.RecordDecision(
			spanCtx,
			event.PaymentEventID,
			event.TenantID,
			event.AggregateAction,
			reasonCodes,
			0.0, // confidence score placeholder — scoring model not yet implemented
			ruleResultsJSON,
			event.LatencyMs,
		)
		if err != nil {
			log.Error("record decision failed", "error", err)
			span.End()
			continue
		}

		c.metrics.DecisionsTotal.Add(spanCtx, 1,
			metric.WithAttributes(attribute.String("outcome", decision.Outcome)))

		if err := c.publisher.PublishDecisionMade(spanCtx, decision); err != nil {
			log.Warn("publish decision.made failed (best-effort)", "error", err)
		} else {
			c.metrics.KafkaPublishTotal.Add(spanCtx, 1)
		}

		c.metrics.DecisionLatency.Record(spanCtx, time.Since(start).Seconds())

		log.Info("decision recorded",
			"decision_id", decision.ID,
			"outcome", decision.Outcome,
			"duration_ms", time.Since(start).Milliseconds())

		span.End()
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

// buildReasonCodes derives human-readable reason codes from matched rules.
func buildReasonCodes(event RiskEvaluatedEvent) []string {
	if event.FailOpen {
		return []string{"fail_open_applied"}
	}
	codes := make([]string, 0)
	for _, r := range event.RuleResults {
		if r.Matched {
			codes = append(codes, r.Action+"_rule:"+r.RuleID)
		}
	}
	if len(codes) == 0 {
		codes = append(codes, "no_rules_matched")
	}
	return codes
}
