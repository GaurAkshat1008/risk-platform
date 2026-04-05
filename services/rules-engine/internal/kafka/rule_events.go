package kafka

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
)

type RuleEvalResult struct {
    RuleID   string `json:"rule_id"`
    RuleName string `json:"rule_name"`
    Matched  bool   `json:"matched"`
    Action   string `json:"action"`
    Reason   string `json:"reason"`
}

type RuleEventPublisher struct {
    producer *Producer
}

func NewRuleEventPublisher(p *Producer) *RuleEventPublisher {
    return &RuleEventPublisher{producer: p}
}

func (p *RuleEventPublisher) PublishRulesEvaluated(ctx context.Context, tenantID, paymentEventID, aggregateAction string, results []RuleEvalResult) error {
    payload := map[string]any{
        "event_type":       "rules.evaluated",
        "tenant_id":        tenantID,
        "payment_event_id": paymentEventID,
        "aggregate_action": aggregateAction,
        "results":          results,
        "occurred_at":      time.Now().UTC(),
    }
    data, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("marshal rules.evaluated: %w", err)
    }
    return p.producer.Publish(ctx, tenantID, data)
}