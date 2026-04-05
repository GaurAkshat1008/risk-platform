package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	GRPCAddr    string
	MetricsAddr string
	LogLevel    slog.Level
	Service     ServiceConfig
	Postgres    PostgresConfig
	Redis       RedisConfig
	Kafka       KafkaConfig
	OTel        OTelConfig
}

type ServiceConfig struct {
	Name string
	Env  string
}

type PostgresConfig struct {
	DSN string
}

type RedisConfig struct {
	Addr string
}

type KafkaConfig struct {
	Brokers       []string
	Topics        []string
	ConsumerGroup string
}

type OTelConfig struct {
	CollectorEndpoint string
}

func Load() (*Config, error) {
	dsn := getEnv("POSTGRES_DSN", "")
	if dsn == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is required")
	}

	brokers := splitTrim(getEnv("KAFKA_BROKERS", "localhost:9092"))
	topics := splitTrim(getEnv("KAFKA_TOPICS", "case.created,case.escalated,decision.made"))

	return &Config{
		GRPCAddr:    getEnv("GRPC_ADDR", ":50061"),
		MetricsAddr: getEnv("METRICS_ADDR", ":9101"),
		LogLevel:    parseLogLevel(getEnv("LOG_LEVEL", "info")),
		Service: ServiceConfig{
			Name: getEnv("OTEL_SERVICE_NAME", "notification"),
			Env:  getEnv("OTEL_ENVIRONMENT", "local"),
		},
		Postgres: PostgresConfig{DSN: dsn},
		Redis:    RedisConfig{Addr: getEnv("REDIS_ADDR", "localhost:6379")},
		Kafka: KafkaConfig{
			Brokers:       brokers,
			Topics:        topics,
			ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "notification-service"),
		},
		OTel: OTelConfig{
			CollectorEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		},
	}, nil
}

func splitTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
