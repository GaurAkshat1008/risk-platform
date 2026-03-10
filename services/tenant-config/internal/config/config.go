package config

import (
	"fmt"
	"log/slog"
	"os"
)

type Config struct {
	GRPCAddr string
	MetricsAddr string
	LogLevel slog.Level
	Service ServiceConfig 
	Postgres PostgresConfig
	Redis RedisConfig
	Kafka KafkaConfig
	OTel OTelConfig
}

type ServiceConfig struct {
	Name string
	Env string
}

type PostgresConfig struct {
	DSN string
}

type RedisConfig struct {
	Addr string
}

type KafkaConfig struct {
	Brokers string
	TennantTopic string
}

type OTelConfig struct {
	CollectorEndpoint string
}

func getEnv(key, defaultValue string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	return v
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func validate(cfg Config) error {
	var errs []error
	if cfg.Postgres.DSN == "" {
		errs = append(errs, fmt.Errorf("POSTGRES_DSN is required"))
	}
	if cfg.Redis.Addr == "" {
		errs = append(errs, fmt.Errorf("REDIS_ADDR is required"))
	}
	if cfg.Kafka.Brokers == "" {
		errs = append(errs, fmt.Errorf("KAFKA_BROKERS is required"))
	}
	if len(errs) > 0 {
		return fmt.Errorf("validation errors: %v", errs)
	}
	return nil
}

func Load() (*Config,error) {
	cfg := &Config{
		GRPCAddr: getEnv("GRPC_ADDR", ":50052"),
		MetricsAddr: getEnv("METRICS_ADDR", ":9091"),
		LogLevel: parseLogLevel(getEnv("LOG_LEVEL", "info")),
		Service: ServiceConfig{
			Name: getEnv("SERVICE_NAME", "tenant-config"),
			Env: getEnv("SERVICE_ENV", "local"),
		},
		Postgres: PostgresConfig{
			DSN: os.Getenv("POSTGRES_DSN"),
		},
		Redis: RedisConfig{
			Addr: getEnv("REDIS_ADDR","localhost:6379"),
		},
		Kafka: KafkaConfig{
			Brokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
			TennantTopic: getEnv("KAFKA_TENANT_TOPIC", "tenant-events"),
		},
		OTel: OTelConfig{
			CollectorEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317"),
		},
	}

	if err := validate(*cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c Config) String() string {
	return fmt.Sprintf("GRPCAddr: %s, MetricsAddr: %s, LogLevel: %s, Service: {Name: %s, Env: %s}, Postgres: {DSN: %s}, Redis: {Addr: %s}, Kafka: {Brokers: %s, TennantTopic: %s}, OTel: {CollectorEndpoint: %s}",
		c.GRPCAddr, c.MetricsAddr, c.LogLevel, c.Service.Name, c.Service.Env, c.Postgres.DSN, c.Redis.Addr, c.Kafka.Brokers, c.Kafka.TennantTopic, c.OTel.CollectorEndpoint)
}