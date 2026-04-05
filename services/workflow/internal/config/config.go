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
	Redis       RedisConfig
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

type OTelConfig struct {
	CollectorEndpoint string
}

func Load() (*Config, error) {
	dsn := getEnv("POSTGRES_DSN", "")
	if dsn == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is required")
	}
	return &Config{
		GRPCAddr:    getEnv("GRPC_ADDR", ":50058"),
		MetricsAddr: getEnv("METRICS_ADDR", ":9098"),
		LogLevel:    parseLogLevel(getEnv("LOG_LEVEL", "info")),
		Service: ServiceConfig{
			Name: getEnv("OTEL_SERVICE_NAME", "workflow"),
			Env:  getEnv("OTEL_ENVIRONMENT", "local"),
		},
		Postgres: PostgresConfig{DSN: dsn},
		Redis:    RedisConfig{Addr: getEnv("REDIS_ADDR", "localhost:6379")},
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
