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
	Upstream    UpstreamConfig
	OTel        OTelConfig
}

type ServiceConfig struct {
	Name string
	Env  string
}

type PostgresConfig struct {
	DSN string
}

// UpstreamConfig holds addresses for gRPC dependencies.
type UpstreamConfig struct {
	DecisionServiceAddr  string
	RulesEngineAddr      string
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
		GRPCAddr:    getEnv("GRPC_ADDR", ":50060"),
		MetricsAddr: getEnv("METRICS_ADDR", ":9100"),
		LogLevel:    parseLogLevel(getEnv("LOG_LEVEL", "info")),
		Service: ServiceConfig{
			Name: getEnv("OTEL_SERVICE_NAME", "explanation"),
			Env:  getEnv("OTEL_ENVIRONMENT", "local"),
		},
		Postgres: PostgresConfig{DSN: dsn},
		Upstream: UpstreamConfig{
			DecisionServiceAddr: getEnv("DECISION_SERVICE_ADDR", "localhost:50056"),
			RulesEngineAddr:     getEnv("RULES_ENGINE_ADDR", "localhost:50055"),
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
