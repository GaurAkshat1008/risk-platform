package config

import (
	"fmt"
	"log/slog"
	"os"
)

type Config struct {
	GRPCAddr    string
	MetricsAddr string
	LogLevel    slog.Level
	Service     ServiceConfig
	Postgres    PostgresConfig
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

type KafkaConfig struct {
	Brokers       string
	LogsTopic     string
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
	return &Config{
		GRPCAddr:    getEnv("GRPC_ADDR", ":50062"),
		MetricsAddr: getEnv("METRICS_ADDR", ":9102"),
		LogLevel:    parseLogLevel(getEnv("LOG_LEVEL", "info")),
		Service: ServiceConfig{
			Name: getEnv("OTEL_SERVICE_NAME", "log-ingestion"),
			Env:  getEnv("OTEL_ENVIRONMENT", "local"),
		},
		Postgres: PostgresConfig{DSN: dsn},
		Kafka: KafkaConfig{
			Brokers:       getEnv("KAFKA_BROKERS", "localhost:9092"),
			LogsTopic:     getEnv("KAFKA_LOGS_TOPIC", "ops.logs"),
			ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "log-ingestion-service"),
		},
		OTel: OTelConfig{
			CollectorEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		},
	}, nil
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
