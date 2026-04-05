package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"audit-trail/internal/config"
	"audit-trail/internal/db"

	kafka "github.com/segmentio/kafka-go"
)

// genericEvent is used to extract common fields from any domain event payload.
type genericEvent struct {
	TenantID       string `json:"tenant_id"`
	ActorID        string `json:"actor_id"`
	Action         string `json:"action"`
	EventType      string `json:"event_type"`
	ResourceType   string `json:"resource_type"`
	ResourceID     string `json:"resource_id"`
	// ID fields from various services
	CaseID         string `json:"case_id"`
	DecisionID     string `json:"decision_id"`
	PaymentEventID string `json:"payment_event_id"`
	ID             string `json:"id"`
}

// Consumer reads from multiple Kafka topics and appends events to the audit store.
type Consumer struct {
	readers []*kafka.Reader
	store   *db.AuditStore
	logger  *slog.Logger
}

// NewConsumer creates one kafka.Reader per topic.
func NewConsumer(cfg config.KafkaConfig, store *db.AuditStore, logger *slog.Logger) *Consumer {
	brokers := strings.Split(cfg.Brokers, ",")
	topics := strings.Split(cfg.Topics, ",")

	readers := make([]*kafka.Reader, 0, len(topics))
	for _, topic := range topics {
		topic = strings.TrimSpace(topic)
		if topic == "" {
			continue
		}
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        cfg.ConsumerGroup,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
			StartOffset:    kafka.FirstOffset,
		})
		readers = append(readers, r)
	}

	return &Consumer{
		readers: readers,
		store:   store,
		logger:  logger,
	}
}

// Start launches one goroutine per topic reader. It blocks until ctx is cancelled.
func (c *Consumer) Start(ctx context.Context) {
	for _, r := range c.readers {
		r := r
		go c.runReader(ctx, r)
	}
}

// Close shuts down all readers.
func (c *Consumer) Close() {
	for _, r := range c.readers {
		_ = r.Close()
	}
}

func (c *Consumer) runReader(ctx context.Context, r *kafka.Reader) {
	topic := r.Config().Topic
	c.logger.Info("audit consumer started", "topic", topic)
	for {
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Error("audit kafka read error", "topic", topic, "error", err)
			continue
		}
		if err := c.processMessage(ctx, msg); err != nil {
			c.logger.Error("audit process message error", "topic", topic, "error", err)
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) error {
	var ev genericEvent
	if err := json.Unmarshal(msg.Value, &ev); err != nil {
		return fmt.Errorf("unmarshal audit event from %s: %w", msg.Topic, err)
	}

	// Derive resource ID from whichever field is present.
	resourceID := ev.ResourceID
	if resourceID == "" {
		switch {
		case ev.CaseID != "":
			resourceID = ev.CaseID
		case ev.DecisionID != "":
			resourceID = ev.DecisionID
		case ev.PaymentEventID != "":
			resourceID = ev.PaymentEventID
		case ev.ID != "":
			resourceID = ev.ID
		}
	}

	// Derive action: prefer explicit action/event_type fields.
	action := ev.Action
	if action == "" {
		action = ev.EventType
	}
	if action == "" {
		action = msg.Topic
	}

	// Derive resource type.
	resourceType := ev.ResourceType
	if resourceType == "" {
		resourceType = topicToResourceType(msg.Topic)
	}

	params := db.AppendEventParams{
		TenantID:     ev.TenantID,
		ActorID:      ev.ActorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		SourceTopic:  msg.Topic,
		Payload:      msg.Value,
	}

	if _, err := c.store.AppendEvent(ctx, params); err != nil {
		return fmt.Errorf("append audit event: %w", err)
	}
	return nil
}

// topicToResourceType maps a Kafka topic name to a resource type string.
func topicToResourceType(topic string) string {
	switch topic {
	case "payments.received":
		return "payment"
	case "risk.evaluated":
		return "risk_assessment"
	case "decision.made":
		return "decision"
	case "case.created", "case.escalated", "case.resolved":
		return "case"
	default:
		return topic
	}
}
