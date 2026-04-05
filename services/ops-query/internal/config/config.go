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
	OTel        OTelConfig
	LogIngestion LogIngestionConfig
	Prometheus   PrometheusConfig
	Jaeger       JaegerConfig
}

type ServiceConfig struct {
	Name string
	Env  string
}

type OTelConfig struct {
	CollectorEndpoint string
}

// LogIngestionConfig holds the address of the log-ingestion gRPC service.
type LogIngestionConfig struct {
	Addr string
}

// PrometheusConfig holds the base HTTP URL for the Prometheus API.
type PrometheusConfig struct {
	Addr string
}

// JaegerConfig holds the base HTTP URL for the Jaeger REST API.
type JaegerConfig struct {
	Addr string
}

func Load() (*Config, error) {
	grpcAddr := getEnv("GRPC_ADDR", ":50063")
	if grpcAddr == "" {
		return nil, fmt.Errorf("GRPC_ADDR must not be empty")
	}
	return &Config{
		GRPCAddr:    grpcAddr,
		MetricsAddr: getEnv("METRICS_ADDR", ":9103"),
		LogLevel:    parseLogLevel(getEnv("LOG_LEVEL", "info")),
		Service: ServiceConfig{
			Name: getEnv("OTEL_SERVICE_NAME", "ops-query"),
			Env:  getEnv("OTEL_ENVIRONMENT", "local"),
		},
		OTel: OTelConfig{
			CollectorEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		},
		LogIngestion: LogIngestionConfig{
			Addr: getEnv("LOG_INGESTION_ADDR", "localhost:50062"),
		},
		Prometheus: PrometheusConfig{
			Addr: getEnv("PROMETHEUS_ADDR", "http://localhost:9090"),
		},
		Jaeger: JaegerConfig{
			Addr: getEnv("JAEGER_ADDR", "http://localhost:16686"),
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
