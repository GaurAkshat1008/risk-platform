package kafka

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "ingestion/internal/db"
)

type PaymentEventPublisher struct {
    producer *Producer
}

func NewPaymentEventPublisher(p *Producer) *PaymentEventPublisher {
    return &PaymentEventPublisher{producer: p}
}

func (p *PaymentEventPublisher) PublishPaymentReceived(ctx context.Context, evt *db.PaymentEvent) error {
    payload := map[string]any{
        "event_type":      "payment.received",
        "event_id":        evt.ID,
        "idempotency_key": evt.IdempotencyKey,
        "tenant_id":       evt.TenantID,
        "amount":          evt.Amount,
        "currency":        evt.Currency,
        "source":          evt.Source,
        "destination":     evt.Destination,
        "received_at":     evt.ReceivedAt.UTC(),
        "occurred_at":     time.Now().UTC(),
    }
    data, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("marshal payment.received: %w", err)
    }
    return p.producer.Publish(ctx, evt.TenantID, data)
}