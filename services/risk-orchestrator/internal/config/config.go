package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

type Config struct {
	GRPCAddr        string
	MetricsAddr     string
	LogLevel        slog.Level
	Service         ServiceConfig
	Redis           RedisConfig
	Kafka           KafkaConfig
	OTel            OTelConfig
	RulesEngineAddr string
	LatencyBudgetMs int64
}

type ServiceConfig struct {
	Name string
	Env  string
}

type RedisConfig struct {
	Addr string
}

type KafkaConfig struct {
	Brokers       string
	PaymentsTopic string
	RiskTopic     string
	ConsumerGroup string
}

type OTelConfig struct {
	CollectorEndpoint string
}

func Load() (*Config, error) {
	budgetMs, err := strconv.ParseInt(getEnv("LATENCY_BUDGET_MS", "150"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid LATENCY_BUDGET_MS: %w", err)
	}

	dsn := getEnv("POSTGRES_DSN", "")
	_ = dsn // no DB for risk-orchestrator — field kept for future use

	return &Config{
		GRPCAddr:    getEnv("GRPC_ADDR", ":50054"),
		MetricsAddr: getEnv("METRICS_ADDR", ":9094"),
		LogLevel:    parseLogLevel(getEnv("LOG_LEVEL", "info")),
		Service: ServiceConfig{
			Name: getEnv("OTEL_SERVICE_NAME", "risk-orchestrator"),
			Env:  getEnv("OTEL_ENVIRONMENT", "local"),
		},
		Redis: RedisConfig{
			Addr: getEnv("REDIS_ADDR", "localhost:6379"),
		},
		Kafka: KafkaConfig{
			Brokers:       getEnv("KAFKA_BROKERS", "localhost:9092"),
			PaymentsTopic: getEnv("KAFKA_PAYMENTS_TOPIC", "payments.received"),
			RiskTopic:     getEnv("KAFKA_RISK_TOPIC", "risk.evaluated"),
			ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "risk-orchestrator"),
		},
		OTel: OTelConfig{
			CollectorEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		},
		RulesEngineAddr: getEnv("RULES_ENGINE_ADDR", "localhost:50055"),
		LatencyBudgetMs: budgetMs,
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
