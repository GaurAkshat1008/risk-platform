package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"graphql-bff/graph"
	"graphql-bff/graph/generated"
	"graphql-bff/internal/auth"
	"graphql-bff/internal/cache"
	"graphql-bff/internal/client"
	"graphql-bff/internal/config"
	"graphql-bff/internal/telemetry"
)

func main() {
	// Load .env (non-fatal if absent in production)
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logLevel := new(slog.LevelVar)
	logLevel.Set(cfg.LogLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── Tracing ──────────────────────────────────────────────────────────────
	tp, err := telemetry.NewTracerProvider(ctx, telemetry.TracerConfig{
		ServiceName:       cfg.Service.Name,
		ServiceVersion:    "1.0.0",
		Environment:       cfg.Service.Env,
		CollectorEndpoint: cfg.OTel.CollectorEndpoint,
	})
	if err != nil {
		logger.Error("failed to create tracer provider", "error", err)
		os.Exit(1)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	// ── Metrics ──────────────────────────────────────────────────────────────
	mp, err := telemetry.NewMeterProvider()
	if err != nil {
		logger.Error("failed to create meter provider", "error", err)
		os.Exit(1)
	}
	defer func() { _ = mp.Shutdown(context.Background()) }()

	metrics, err := telemetry.NewMetrics(mp)
	if err != nil {
		logger.Error("failed to create metrics", "error", err)
		os.Exit(1)
	}

	// ── Redis cache ───────────────────────────────────────────────────────────
	queryCache := cache.NewQueryCache(cfg.Redis.Addr)

	// ── gRPC client registry ──────────────────────────────────────────────────
	registry, err := client.NewRegistry(client.Addrs{
		Decision:     cfg.Backends.DecisionAddr,
		CaseMgmt:     cfg.Backends.CaseManagementAddr,
		Ingestion:    cfg.Backends.IngestionAddr,
		Explanation:  cfg.Backends.ExplanationAddr,
		Workflow:     cfg.Backends.WorkflowAddr,
		Audit:        cfg.Backends.AuditTrailAddr,
		Notification: cfg.Backends.NotificationAddr,
		OpsQuery:     cfg.Backends.OpsQueryAddr,
		Rules:        cfg.Backends.RulesEngineAddr,
		Tenant:       cfg.Backends.TenantConfigAddr,
		Identity:     cfg.Backends.IdentityAccessAddr,
	})
	if err != nil {
		logger.Error("failed to create client registry", "error", err)
		os.Exit(1)
	}
	defer registry.Close()

	// ── GraphQL server ────────────────────────────────────────────────────────
	resolver := &graph.Resolver{
		Clients: registry,
		Cache:   queryCache,
		Logger:  logger,
		Metrics: metrics,
	}

	gqlServer := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// ── HTTP routing ──────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	graphqlHandler := auth.Middleware(
		telemetry.RequestTimer(metrics,
			otelhttp.NewHandler(gqlServer, "graphql"),
		),
	)
	mux.Handle("/graphql", graphqlHandler)
	mux.Handle("/playground", playground.Handler("GraphQL Playground", "/graphql"))
	mux.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	// Metrics server (separate port)
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{
		Addr:         cfg.Server.MetricsAddr,
		Handler:      metricsMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() {
		logger.Info("metrics server listening", "addr", cfg.Server.MetricsAddr)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server error", "error", err)
		}
	}()

	// Main server
	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("graphql-bff listening", "addr", cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	_ = metricsServer.Shutdown(shutdownCtx)
}
