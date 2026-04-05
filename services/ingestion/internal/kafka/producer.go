package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
	logger *slog.Logger
}

type Config struct {
	Brokers []string
	Topic string
}


func NewProducer(cfg Config, logger *slog.Logger) *Producer {
	writer := &kafka.Writer{
		Addr: kafka.TCP(cfg.Brokers...),
		Topic: cfg.Topic,
		Balancer: &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async: true,
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			logger.Error(fmt.Sprintf(msg, args...))
		}),
}
	return &Producer{
		writer: writer,
		logger: logger,
	}
}

func (p *Producer) Publish(ctx context.Context, key string, value []byte) error {
    msg := kafka.Message{
        Key:   []byte(key),
        Value: value,
        Time:  time.Now(),
    }
    if err := p.writer.WriteMessages(ctx, msg); err != nil {
        return fmt.Errorf("kafka write: %w", err)
    }
    return nil
}

func (p *Producer) Close() {
    if err := p.writer.Close(); err != nil {
        p.logger.Error("kafka writer close error", "error", err)
    }
}