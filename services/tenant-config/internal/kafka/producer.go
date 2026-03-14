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
    Topic   string
}

func NewProducer(cfg Config, logger *slog.Logger) *Producer {
    w := &kafka.Writer{
        Addr:         kafka.TCP(cfg.Brokers...),
        Topic:        cfg.Topic,
        Balancer:     &kafka.LeastBytes{},
        RequiredAcks: kafka.RequireOne,
        Async:        true,
        ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
            logger.Error(fmt.Sprintf(msg, args...))
        }),
    }
    return &Producer{writer: w, logger: logger}
}

func (p *Producer) Publish(ctx context.Context, key string, value []byte) error {
    return p.writer.WriteMessages(ctx, kafka.Message{
        Key:   []byte(key),
        Value: value,
        Time:  time.Now(),
    })
}

func (p *Producer) Close() error {
    return p.writer.Close()
}