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
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}){
			logger.Error(fmt.Sprintf(msg, args...))
		}),
	}

	return &Producer {
		writer: writer,
		logger: logger,
	}
}

func (p *Producer) Publish(ctx context.Context, key string, value []byte) error {
	msg := kafka.Message{
		Key: []byte(key),
		Value: value,
		Time: time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	return nil
}

func (p *Producer) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("close producer: %w", err)
	}
	return nil	
}

func (p *Producer) Ping(ctx context.Context) error {
	conn, err := kafka.DialContext(ctx, "tcp", p.writer.Addr.String())
	if err != nil {
		return fmt.Errorf("dial kafka: %w", err)
	}
	defer conn.Close()

	return nil
}