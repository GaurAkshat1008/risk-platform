package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"notification/internal/cache"
	"notification/internal/db"

	kafka "github.com/segmentio/kafka-go"
)

// genericEvent extracts the common fields shared across all consumed domain events.
type genericEvent struct {
	TenantID      string `json:"tenant_id"`
	EventType     string `json:"event_type"`
	CaseID        string `json:"case_id"`
	DecisionID    string `json:"decision_id"`
	PaymentEventID string `json:"payment_event_id"`
	ID            string `json:"id"`
}

// Consumer subscribes to multiple Kafka topics and creates notification records.
type Consumer struct {
	readers     []*kafka.Reader
	store       *db.NotificationStore
	rateLimiter *cache.RateLimiter
	logger      *slog.Logger
}

// NewConsumer creates one kafka.Reader per topic in cfg.Topics.
func NewConsumer(brokers, topics []string, groupID string,
	store *db.NotificationStore, rateLimiter *cache.RateLimiter, logger *slog.Logger,
) *Consumer {
	readers := make([]*kafka.Reader, 0, len(topics))
	for _, topic := range topics {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
			StartOffset:    kafka.FirstOffset,
		})
		readers = append(readers, r)
	}
	return &Consumer{
		readers:     readers,
		store:       store,
		rateLimiter: rateLimiter,
		logger:      logger,
	}
}

// Start launches one goroutine per topic reader; blocks until ctx is cancelled.
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
	c.logger.Info("notification consumer started", "topic", topic)
	for {
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Error("kafka read error", "topic", topic, "error", err)
			continue
		}
		if err := c.processMessage(ctx, msg); err != nil {
			c.logger.Error("process message error", "topic", topic, "error", err)
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) error {
	var ev genericEvent
	if err := json.Unmarshal(msg.Value, &ev); err != nil {
		return fmt.Errorf("unmarshal event from %s: %w", msg.Topic, err)
	}

	if ev.TenantID == "" {
		c.logger.Warn("dropping event with empty tenant_id", "topic", msg.Topic)
		return nil
	}

	// Check rate limit for this tenant before creating a notification.
	allowed, err := c.rateLimiter.Allow(ctx, ev.TenantID, "all")
	if err != nil {
		c.logger.Warn("rate limiter error, allowing by default", "error", err)
		allowed = true
	}
	if !allowed {
		c.logger.Info("rate limit hit, dropping notification",
			"tenant_id", ev.TenantID, "topic", msg.Topic)
		return nil
	}

	// Determine the resource identifier for the recipient field (used for routing).
	resourceID := ev.CaseID
	if resourceID == "" {
		resourceID = ev.DecisionID
	}
	if resourceID == "" {
		resourceID = ev.PaymentEventID
	}
	if resourceID == "" {
		resourceID = ev.ID
	}

	n := &db.Notification{
		TenantID:  ev.TenantID,
		Type:      msg.Topic,
		Recipient: fmt.Sprintf("tenant:%s:resource:%s", ev.TenantID, resourceID),
		Channel:   "webhook", // default channel; preferences override at delivery time
		Payload:   string(msg.Value),
	}
	id, err := c.store.CreateNotification(ctx, n)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	c.logger.Info("notification queued",
		"notification_id", id, "tenant_id", ev.TenantID, "topic", msg.Topic)
	return nil
}
