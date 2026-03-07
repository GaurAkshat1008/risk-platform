package kafka

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
)

// EventType defines the type of auth event
type EventType string

const (
    EventTokenValidated   EventType = "token.validated"
    EventTokenRejected    EventType = "token.rejected"
    EventAuthorizeAllowed EventType = "authorize.allowed"
    EventAuthorizeDenied  EventType = "authorize.denied"
    EventAccountDisabled  EventType = "account.disabled_detected"
    EventTenantMismatch   EventType = "tenant.mismatch_detected"
)

// AuthEvent is the structure published to Kafka
type AuthEvent struct {
    EventID    string    `json:"event_id"`
    EventType  EventType `json:"event_type"`
    UserID     string    `json:"user_id"`
    TenantID   string    `json:"tenant_id"`
    Action     string    `json:"action,omitempty"`
    Allowed    bool      `json:"allowed,omitempty"`
    Reason     string    `json:"reason,omitempty"`
    TraceID    string    `json:"trace_id,omitempty"`
    OccurredAt time.Time `json:"occurred_at"`
}

type AuthEventPublisher struct {
    producer *Producer
}

func NewAuthEventPublisher(producer *Producer) *AuthEventPublisher {
    return &AuthEventPublisher{producer: producer}
}

func (p *AuthEventPublisher) PublishTokenValidated(ctx context.Context, userID, tenantID, traceID string) error {
    return p.publish(ctx, AuthEvent{
        EventType:  EventTokenValidated,
        UserID:     userID,
        TenantID:   tenantID,
        TraceID:    traceID,
        OccurredAt: time.Now(),
    })
}

func (p *AuthEventPublisher) PublishTokenRejected(ctx context.Context, reason, traceID string) error {
    return p.publish(ctx, AuthEvent{
        EventType:  EventTokenRejected,
        Reason:     reason,
        TraceID:    traceID,
        OccurredAt: time.Now(),
    })
}

func (p *AuthEventPublisher) PublishAuthzDecision(ctx context.Context, userID, tenantID, action string, allowed bool, reason, traceID string) error {
    eventType := EventAuthorizeAllowed
    if !allowed {
        eventType = EventAuthorizeDenied
    }

    return p.publish(ctx, AuthEvent{
        EventType:  eventType,
        UserID:     userID,
        TenantID:   tenantID,
        Action:     action,
        Allowed:    allowed,
        Reason:     reason,
        TraceID:    traceID,
        OccurredAt: time.Now(),
    })
}

func (p *AuthEventPublisher) PublishAccountDisabled(ctx context.Context, userID, tenantID, traceID string) error {
    return p.publish(ctx, AuthEvent{
        EventType:  EventAccountDisabled,
        UserID:     userID,
        TenantID:   tenantID,
        TraceID:    traceID,
        OccurredAt: time.Now(),
    })
}

func (p *AuthEventPublisher) publish(ctx context.Context, event AuthEvent) error {
    event.EventID = fmt.Sprintf("%s-%d", event.EventType, time.Now().UnixNano())

    payload, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("marshal auth event failed: %w", err)
    }

    return p.producer.Publish(ctx, event.UserID, payload)
}