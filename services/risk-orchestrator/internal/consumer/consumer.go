package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"risk-orchestrator/internal/kafka"
	"risk-orchestrator/internal/orchestrator"

	kafkago "github.com/segmentio/kafka-go"
)

// publisherAdapter wraps *kafka.RiskEventPublisher and implements orchestrator.Publisher.
// It lives here — in the leaf package — so neither kafka nor orchestrator imports the other.
type publisherAdapter struct {
	p *kafka.RiskEventPublisher
}

// NewPublisherAdapter wraps a Kafka publisher so it satisfies orchestrator.Publisher.
func NewPublisherAdapter(p *kafka.RiskEventPublisher) orchestrator.Publisher {
	return &publisherAdapter{p: p}
}

func (a *publisherAdapter) PublishRiskEvaluated(ctx context.Context, payload orchestrator.RiskEvaluatedPayload) error {
	summaries := make([]kafka.RuleResultSummary, 0, len(payload.RuleResults))
	for _, r := range payload.RuleResults {
		summaries = append(summaries, kafka.RuleResultSummary{
			RuleID:   r.RuleID,
			RuleName: r.RuleName,
			Matched:  r.Matched,
			Action:   r.Action,
			Reason:   r.Reason,
		})
	}
	return a.p.PublishRiskEvaluated(ctx, kafka.RiskEvaluatedEvent{
		PaymentEventID:  payload.PaymentEventID,
		TenantID:        payload.TenantID,
		AggregateAction: payload.AggregateAction,
		FailOpen:        payload.FailOpen,
		RuleResults:     summaries,
		LatencyMs:       payload.LatencyMs,
		EvaluatedAt:     payload.EvaluatedAt,
	})
}

// ── Kafka Consumer ───────────────────────────────────────────────────────────

// paymentReceivedEvent is the schema of messages on the payments.received topic.
type paymentReceivedEvent struct {
	EventType      string    `json:"event_type"`
	EventID        string    `json:"event_id"`
	TenantID       string    `json:"tenant_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	Source         string    `json:"source"`
	Destination    string    `json:"destination"`
	ReceivedAt     time.Time `json:"received_at"`
	OccurredAt     time.Time `json:"occurred_at"`
}

// ConsumerConfig holds Kafka consumer connection settings.
type ConsumerConfig struct {
	Brokers       string
	PaymentsTopic string
	ConsumerGroup string
}

// Consumer reads payment events from Kafka and drives the risk orchestration pipeline.
type Consumer struct {
	reader *kafkago.Reader
	orch   *orchestrator.Orchestrator
	logger *slog.Logger
}

// NewConsumer creates a Consumer that calls orch.Process for every payment event.
func NewConsumer(cfg ConsumerConfig, orch *orchestrator.Orchestrator, logger *slog.Logger) *Consumer {
	r := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:        strings.Split(cfg.Brokers, ","),
		Topic:          cfg.PaymentsTopic,
		GroupID:        cfg.ConsumerGroup,
		MinBytes:       1,
		MaxBytes:       10 << 20, // 10 MB
		CommitInterval: 0,
	})
	return &Consumer{reader: r, orch: orch, logger: logger}
}

// Run blocks, consuming messages until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) {
	c.logger.Info("kafka consumer started", "topic", c.reader.Config().Topic)
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.logger.Info("kafka consumer stopping")
				return
			}
			c.logger.Error("kafka read error", "error", err)
			continue
		}

		var event paymentReceivedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.Error("unmarshal payment.received failed",
				"partition", msg.Partition, "offset", msg.Offset, "error", err)
			continue
		}

		if err := c.orch.Process(ctx, orchestrator.PaymentEvent{
			PaymentEventID: event.EventID,
			TenantID:       event.TenantID,
			Amount:         int64(event.Amount),
			Currency:       event.Currency,
			Source:         event.Source,
			Destination:    event.Destination,
			Metadata:       map[string]string{},
		}); err != nil {
			c.logger.Error("orchestration failed",
				"payment_event_id", event.EventID,
				"tenant_id", event.TenantID,
				"error", err)
		}
	}
}

// Close shuts down the Kafka reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}
