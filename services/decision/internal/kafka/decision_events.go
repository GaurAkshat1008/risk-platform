package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"decision/internal/db"
)

// DecisionEventPublisher publishes decision.made events to Kafka.
type DecisionEventPublisher struct {
	producer *Producer
}

func NewDecisionEventPublisher(p *Producer) *DecisionEventPublisher {
	return &DecisionEventPublisher{producer: p}
}

func (p *DecisionEventPublisher) PublishDecisionMade(ctx context.Context, d *db.Decision) error {
	payload := map[string]any{
		"event_type":       "decision.made",
		"decision_id":      d.ID,
		"payment_event_id": d.PaymentEventID,
		"tenant_id":        d.TenantID,
		"outcome":          d.Outcome,
		"reason_codes":     d.ReasonCodes,
		"confidence_score": d.ConfidenceScore,
		"latency_ms":       d.LatencyMs,
		"overridden":       d.Overridden,
		"occurred_at":      time.Now().UTC(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal decision.made: %w", err)
	}
	return p.producer.Publish(ctx, d.TenantID, data)
}
