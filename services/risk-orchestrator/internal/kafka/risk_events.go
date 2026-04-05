package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// RuleResultSummary is a per-rule outcome included in the risk.evaluated event.
type RuleResultSummary struct {
	RuleID   string `json:"rule_id"`
	RuleName string `json:"rule_name"`
	Matched  bool   `json:"matched"`
	Action   string `json:"action"`
	Reason   string `json:"reason"`
}

// RiskEvaluatedEvent is published to the risk.evaluated Kafka topic.
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

type RiskEventPublisher struct {
	producer *Producer
}

func NewRiskEventPublisher(p *Producer) *RiskEventPublisher {
	return &RiskEventPublisher{producer: p}
}

func (p *RiskEventPublisher) PublishRiskEvaluated(ctx context.Context, event RiskEvaluatedEvent) error {
	event.EventType = "risk.evaluated"
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal risk.evaluated: %w", err)
	}
	return p.producer.Publish(ctx, event.TenantID, data)
}
