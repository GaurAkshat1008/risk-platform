package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Server   ServerConfig
	Service  ServiceConfig
	OTel     OTelConfig
	Redis    RedisConfig
	Backends BackendConfig
	LogLevel slog.Level
}

type ServerConfig struct {
	Addr        string // GraphQL HTTP server addr
	MetricsAddr string
	CORSOrigins []string
}

type ServiceConfig struct {
	Name string
	Env  string
}

type OTelConfig struct {
	CollectorEndpoint string
}

type RedisConfig struct {
	Addr string
}

// BackendConfig holds gRPC addresses for all downstream services.
type BackendConfig struct {
	IdentityAccessAddr  string
	TenantConfigAddr    string
	IngestionAddr       string
	RulesEngineAddr     string
	RiskOrchestratorAddr string
	DecisionAddr        string
	CaseManagementAddr  string
	WorkflowAddr        string
	AuditTrailAddr      string
	ExplanationAddr     string
	NotificationAddr    string
	LogIngestionAddr    string
	OpsQueryAddr        string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Addr:        env("HTTP_ADDR", ":8090"),
			MetricsAddr: env("METRICS_ADDR", ":9104"),
			CORSOrigins: splitTrim(env("CORS_ORIGINS", "*")),
		},
		Service: ServiceConfig{
			Name: env("SERVICE_NAME", "graphql-bff"),
			Env:  env("ENVIRONMENT", "development"),
		},
		OTel: OTelConfig{
			CollectorEndpoint: env("OTEL_COLLECTOR_ENDPOINT", "localhost:4317"),
		},
		Redis: RedisConfig{
			Addr: env("REDIS_ADDR", "localhost:6379"),
		},
		Backends: BackendConfig{
			IdentityAccessAddr:   env("IDENTITY_ACCESS_ADDR", "localhost:50051"),
			TenantConfigAddr:     env("TENANT_CONFIG_ADDR", "localhost:50052"),
			IngestionAddr:        env("INGESTION_ADDR", "localhost:50053"),
			RiskOrchestratorAddr: env("RISK_ORCHESTRATOR_ADDR", "localhost:50054"),
			RulesEngineAddr:      env("RULES_ENGINE_ADDR", "localhost:50055"),
			DecisionAddr:         env("DECISION_ADDR", "localhost:50056"),
			CaseManagementAddr:   env("CASE_MANAGEMENT_ADDR", "localhost:50057"),
			WorkflowAddr:         env("WORKFLOW_ADDR", "localhost:50058"),
			AuditTrailAddr:       env("AUDIT_TRAIL_ADDR", "localhost:50059"),
			ExplanationAddr:      env("EXPLANATION_ADDR", "localhost:50060"),
			NotificationAddr:     env("NOTIFICATION_ADDR", "localhost:50061"),
			LogIngestionAddr:     env("LOG_INGESTION_ADDR", "localhost:50062"),
			OpsQueryAddr:         env("OPS_QUERY_ADDR", "localhost:50063"),
		},
		LogLevel: parseLogLevel(env("LOG_LEVEL", "info")),
	}

	if cfg.Server.Addr == "" {
		return nil, fmt.Errorf("HTTP_ADDR is required")
	}

	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
