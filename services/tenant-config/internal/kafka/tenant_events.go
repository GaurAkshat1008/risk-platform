package kafka

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
)

type TenantEventPublisher struct {
    producer *Producer
}

func NewTenantEventPublisher(p *Producer) *TenantEventPublisher {
    return &TenantEventPublisher{producer: p}
}

func (p *TenantEventPublisher) PublishTenantCreated(ctx context.Context, tenantID, tenantName string) error {
    evt := map[string]any{
        "event_type":  "tenant.created",
        "tenant_id":   tenantID,
        "tenant_name": tenantName,
        "occurred_at": time.Now().UTC(),
    }
    data, err := json.Marshal(evt)
    if err != nil {
        return fmt.Errorf("marshal tenant.created: %w", err)
    }
    return p.producer.Publish(ctx, tenantID, data)
}

func (p *TenantEventPublisher) PublishConfigUpdated(ctx context.Context, tenantID, changeType string, version int32) error {
    evt := map[string]any{
        "event_type":  "tenant.config.updated",
        "tenant_id":   tenantID,
        "change_type": changeType,
        "version":     version,
        "occurred_at": time.Now().UTC(),
    }
    data, err := json.Marshal(evt)
    if err != nil {
        return fmt.Errorf("marshal tenant.config.updated: %w", err)
    }
    return p.producer.Publish(ctx, tenantID, data)
}