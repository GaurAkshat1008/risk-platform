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
	RiskTopic     string
	DecisionTopic string
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
		GRPCAddr:    getEnv("GRPC_ADDR", ":50056"),
		MetricsAddr: getEnv("METRICS_ADDR", ":9096"),
		LogLevel:    parseLogLevel(getEnv("LOG_LEVEL", "info")),
		Service: ServiceConfig{
			Name: getEnv("OTEL_SERVICE_NAME", "decision"),
			Env:  getEnv("OTEL_ENVIRONMENT", "local"),
		},
		Postgres: PostgresConfig{DSN: dsn},
		Kafka: KafkaConfig{
			Brokers:       getEnv("KAFKA_BROKERS", "localhost:9092"),
			RiskTopic:     getEnv("KAFKA_RISK_TOPIC", "risk.evaluated"),
			DecisionTopic: getEnv("KAFKA_DECISION_TOPIC", "decision.made"),
			ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "decision-service"),
		},
		OTel: OTelConfig{
			CollectorEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		},
	}, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseLogLevel(level string) slog.Level {
	switch level {
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
