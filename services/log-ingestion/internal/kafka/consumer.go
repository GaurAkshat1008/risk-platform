package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"log-ingestion/internal/config"
	"log-ingestion/internal/db"

	kafka "github.com/segmentio/kafka-go"
)

// logMessage is the expected JSON shape of messages on the ops.logs topic.
// Fields map to what OTel Collector emits when forwarding structured log records.
type logMessage struct {
	Service     string                 `json:"service"`
	Severity    string                 `json:"severity"`
	Level       string                 `json:"level"`       // alias used by some collectors
	SeverityText string                `json:"severityText"` // OTel OTLP JSON encoding
	Body        string                 `json:"body"`        // OTel log body
	Message     string                 `json:"message"`     // alternative field name
	TraceID     string                 `json:"trace_id"`
	SpanID      string                 `json:"span_id"`
	TenantID    string                 `json:"tenant_id"`
	Environment string                 `json:"environment"`
	Timestamp   string                 `json:"timestamp"`   // RFC3339
	TimeUnixNano int64                 `json:"timeUnixNano"`
	Attributes  map[string]interface{} `json:"attributes"`
	Resource    map[string]interface{} `json:"resource"` // OTel resource attributes
}

// Consumer reads log events from the ops.logs Kafka topic and stores them.
type Consumer struct {
	reader *kafka.Reader
	store  *db.LogStore
	logger *slog.Logger
}

func NewConsumer(cfg config.KafkaConfig, store *db.LogStore, logger *slog.Logger) *Consumer {
	brokers := strings.Split(cfg.Brokers, ",")
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          cfg.LogsTopic,
		GroupID:        cfg.ConsumerGroup,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,
	})
	return &Consumer{reader: r, store: store, logger: logger}
}

// Start begins consuming from ops.logs. Blocks until ctx is cancelled.
func (c *Consumer) Start(ctx context.Context) {
	go c.run(ctx)
}

func (c *Consumer) Close() {
	_ = c.reader.Close()
}

func (c *Consumer) run(ctx context.Context) {
	c.logger.Info("log consumer started", "topic", c.reader.Config().Topic)
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Error("kafka read error", "error", err)
			continue
		}
		if err := c.process(ctx, msg); err != nil {
			c.logger.Error("process log message error", "error", err)
		}
	}
}

func (c *Consumer) process(ctx context.Context, msg kafka.Message) error {
	var lm logMessage
	if err := json.Unmarshal(msg.Value, &lm); err != nil {
		return fmt.Errorf("unmarshal log message: %w", err)
	}

	// Normalise severity — accept multiple field names.
	severity := normaliseSeverity(coalesce(lm.SeverityText, lm.Severity, lm.Level))

	// Normalise message body — accept multiple field names.
	message := coalesce(lm.Body, lm.Message)

	// Normalise service — can also live in the resource map.
	service := lm.Service
	if service == "" {
		if lm.Resource != nil {
			if sv, ok := lm.Resource["service.name"].(string); ok {
				service = sv
			}
		}
	}
	if service == "" {
		service = "unknown"
	}

	// Normalise timestamp.
	ts := msg.Time
	if lm.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, lm.Timestamp); err == nil {
			ts = parsed
		}
	} else if lm.TimeUnixNano > 0 {
		ts = time.Unix(0, lm.TimeUnixNano).UTC()
	}

	// Re-serialise attributes map as JSON bytes.
	var attrBytes []byte
	if lm.Attributes != nil {
		b, err := json.Marshal(lm.Attributes)
		if err == nil {
			attrBytes = b
		}
	}

	params := db.InsertParams{
		Service:     service,
		Severity:    severity,
		Message:     message,
		TraceID:     lm.TraceID,
		SpanID:      lm.SpanID,
		TenantID:    lm.TenantID,
		Environment: lm.Environment,
		Attributes:  attrBytes,
		Timestamp:   ts,
	}

	if _, err := c.store.Insert(ctx, params); err != nil {
		return fmt.Errorf("store log entry: %w", err)
	}
	return nil
}

// normaliseSeverity maps common level strings to our canonical uppercase set.
func normaliseSeverity(s string) string {
	switch strings.ToUpper(s) {
	case "DEBUG", "TRACE":
		return "DEBUG"
	case "INFO", "INFORMATION":
		return "INFO"
	case "WARN", "WARNING":
		return "WARN"
	case "ERROR", "ERR":
		return "ERROR"
	case "FATAL", "CRITICAL", "PANIC":
		return "FATAL"
	default:
		return "INFO"
	}
}

// coalesce returns the first non-empty string from the arguments.
func coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
