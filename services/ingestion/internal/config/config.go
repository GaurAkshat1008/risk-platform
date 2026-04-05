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
    Brokers       string
    PaymentsTopic string
}

type OTelConfig struct {
    CollectorEndpoint string
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

func Load() (Config, error) {
    cfg := Config{
        GRPCAddr:    getEnv("GRPC_ADDR", ":50053"),
        MetricsAddr: getEnv("METRICS_ADDR", ":9093"),
        LogLevel:    parseLogLevel(getEnv("LOG_LEVEL", "info")),
        Service: ServiceConfig{
            Name: getEnv("SERVICE_NAME", "ingestion"),
            Env:  getEnv("SERVICE_ENV", "local"),
        },
        Postgres: PostgresConfig{DSN: getEnv("POSTGRES_DSN", "")},
        Redis:    RedisConfig{Addr: getEnv("REDIS_ADDR", "")},
        Kafka: KafkaConfig{
            Brokers:       getEnv("KAFKA_BROKERS", ""),
            PaymentsTopic: getEnv("KAFKA_PAYMENTS_TOPIC", "payments.received"),
        },
        OTel: OTelConfig{
            CollectorEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
        },
    }
    var errs []string
    if cfg.Postgres.DSN == "" {
        errs = append(errs, "POSTGRES_DSN is required")
    }
    if cfg.Redis.Addr == "" {
        errs = append(errs, "REDIS_ADDR is required")
    }
    if cfg.Kafka.Brokers == "" {
        errs = append(errs, "KAFKA_BROKERS is required")
    }
    if len(errs) > 0 {
        return Config{}, fmt.Errorf("config validation: %v", errs)
    }
    return cfg, nil
}