package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"case-management/internal/db"
)

// CaseEventPublisher publishes case lifecycle events to Kafka.
type CaseEventPublisher struct {
	producer *Producer
}

func NewCaseEventPublisher(p *Producer) *CaseEventPublisher {
	return &CaseEventPublisher{producer: p}
}

// PublishCaseCreated publishes a case.created event.
func (p *CaseEventPublisher) PublishCaseCreated(ctx context.Context, c *db.Case) error {
	payload, err := json.Marshal(map[string]any{
		"event_type":       "case.created",
		"case_id":          c.ID,
		"decision_id":      c.DecisionID,
		"tenant_id":        c.TenantID,
		"payment_event_id": c.PaymentEventID,
		"outcome":          c.Outcome,
		"priority":         c.Priority,
		"status":           c.Status,
		"sla_deadline":     c.SLADeadline.UTC(),
		"occurred_at":      time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("marshal case.created: %w", err)
	}
	return p.producer.Publish(ctx, c.TenantID, payload)
}

// PublishCaseEscalated publishes a case.escalated event.
func (p *CaseEventPublisher) PublishCaseEscalated(ctx context.Context, c *db.Case, reason string) error {
	payload, err := json.Marshal(map[string]any{
		"event_type":  "case.escalated",
		"case_id":     c.ID,
		"tenant_id":   c.TenantID,
		"reason":      reason,
		"occurred_at": time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("marshal case.escalated: %w", err)
	}
	return p.producer.Publish(ctx, c.TenantID, payload)
}

// PublishCaseResolved publishes a case.resolved event.
func (p *CaseEventPublisher) PublishCaseResolved(ctx context.Context, c *db.Case, actorID string) error {
	payload, err := json.Marshal(map[string]any{
		"event_type":  "case.resolved",
		"case_id":     c.ID,
		"tenant_id":   c.TenantID,
		"actor_id":    actorID,
		"occurred_at": time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("marshal case.resolved: %w", err)
	}
	return p.producer.Publish(ctx, c.TenantID, payload)
}
