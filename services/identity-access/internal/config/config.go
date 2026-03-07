package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
)

type Config struct {
	GRPCAddr string
	Kafka KafkaConfig
	Keycloak KeycloakConfig
	LogLevel slog.Level
	MetricsAddr string
	OTel OTelConfig
	Service  ServiceConfig
}

type ServiceConfig struct {
	Name string
	Env  string
}

type KeycloakConfig struct {
	BaseURL      string
	Realm        string
	ClientID     string
	ClientSecret string
	Issuer       string
	Audience     string
}

type OTelConfig struct {
    CollectorEndpoint string
}

type KafkaConfig struct {
    Brokers         string
    AuthEventsTopic string
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

    if cfg.GRPCAddr == "" {
        errs = append(errs, errors.New("GRPC_ADDR is required"))
    }
    if cfg.Keycloak.BaseURL == "" {
        errs = append(errs, errors.New("KEYCLOAK_BASE_URL is required"))
    }
    if cfg.Keycloak.Realm == "" {
        errs = append(errs, errors.New("KEYCLOAK_REALM is required"))
    }
    if cfg.Keycloak.ClientID == "" {
        errs = append(errs, errors.New("KEYCLOAK_CLIENT_ID is required"))
    }
    if cfg.Keycloak.ClientSecret == "" {
        errs = append(errs, errors.New("KEYCLOAK_CLIENT_SECRET is required"))
    }
    if cfg.Keycloak.Issuer == "" {
        errs = append(errs, errors.New("KEYCLOAK_ISSUER is required"))
    }
    if cfg.Keycloak.Audience == "" {
        errs = append(errs, errors.New("KEYCLOAK_AUDIENCE is required"))
    }

    if len(errs) > 0 {
        return errors.Join(errs...)
    }
    return nil
}

func Load() (Config, error) {
	 cfg := Config{
        GRPCAddr:    getEnv("GRPC_ADDR", ":50051"),
        MetricsAddr: getEnv("METRICS_ADDR", ":9090"),
        LogLevel:    parseLogLevel(getEnv("LOG_LEVEL", "INFO")),
        Service: ServiceConfig{
            Name: getEnv("SERVICE_NAME", "identity-access"),
            Env:  getEnv("SERVICE_ENV", "local"),
        },
        Keycloak: KeycloakConfig{
            BaseURL:      os.Getenv("KEYCLOAK_BASE_URL"),
            Realm:        os.Getenv("KEYCLOAK_REALM"),
            ClientID:     os.Getenv("KEYCLOAK_CLIENT_ID"),
            ClientSecret: os.Getenv("KEYCLOAK_CLIENT_SECRET"),
            Issuer:       os.Getenv("KEYCLOAK_ISSUER"),
            Audience:     os.Getenv("KEYCLOAK_AUDIENCE"),
        },
        OTel: OTelConfig{
            CollectorEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
        },
        Kafka: KafkaConfig{
            Brokers:         getEnv("KAFKA_BROKERS", "localhost:9092"),
            AuthEventsTopic: getEnv("KAFKA_AUTH_EVENTS_TOPIC", "auth.events"),
        },
	}

	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) String() string {
	return fmt.Sprintf(
		"GRPCAddr: %s, LogLevel: %s, Service: {Name: %s, Env: %s}, Keycloak: {BaseURL: %s, Realm: %s, ClientID: %s, Issuer: %s, Audience: %s}",
		c.GRPCAddr,
		c.LogLevel,
		c.Service.Name,
		c.Service.Env,
		c.Keycloak.BaseURL,
		c.Keycloak.Realm,
		c.Keycloak.ClientID,
		c.Keycloak.Issuer,
		c.Keycloak.Audience,
	)
}