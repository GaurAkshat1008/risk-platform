package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	audittrailpb "audit-trail/api/gen/audit-trail"
	"audit-trail/internal/config"
	"audit-trail/internal/db"
	auditgrpc "audit-trail/internal/grpc"
	"audit-trail/internal/kafka"
	"audit-trail/internal/telemetry"
	"audit-trail/migrations"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load environment variables from .env if present.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── Tracing ──────────────────────────────────────────────────────────────
	tp, err := telemetry.NewTracerProvider(ctx, telemetry.TracerConfig{
		OTLPEndpoint: cfg.OTel.CollectorEndpoint,
		ServiceName:  cfg.Service.Name,
		Environment:  cfg.Service.Env,
	})
	if err != nil {
		logger.Error("init tracer", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		_ = tp.Shutdown(shutCtx)
	}()

	// ── Metrics ───────────────────────────────────────────────────────────────
	promExporter, err := prometheus.New()
	if err != nil {
		logger.Error("init prometheus exporter", "error", err)
		os.Exit(1)
	}
	metrics, err := telemetry.NewMetrics(ctx, sdkmetric.NewManualReader())
	if err != nil {
		logger.Error("init metrics", "error", err)
		os.Exit(1)
	}
	_ = promExporter // registered globally by the prometheus exporter

	// ── Database ──────────────────────────────────────────────────────────────
	pool, err := db.NewPool(ctx, cfg.Postgres.DSN)
	if err != nil {
		logger.Error("init db pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// ── Migrations ───────────────────────────────────────────────────────────
	if err := migrations.Run(ctx, pool); err != nil {
		logger.Error("run migrations", "error", err)
		os.Exit(1)
	}
	logger.Info("migrations applied")

	// ── Store ─────────────────────────────────────────────────────────────────
	store := db.NewAuditStore(pool)

	// ── Kafka consumer (multi-topic) ──────────────────────────────────────────
	consumer := kafka.NewConsumer(cfg.Kafka, store, logger)
	consumer.Start(ctx)
	defer consumer.Close()

	// ── gRPC server ───────────────────────────────────────────────────────────
	auditSvc := auditgrpc.NewAuditService(store, metrics, logger)

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			telemetry.UnaryServerInterceptor(metrics),
		),
	)

	audittrailpb.RegisterAuditTrailServiceServer(grpcServer, auditSvc)

	healthSvc := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSvc)
	healthSvc.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	// ── Metrics HTTP endpoint ─────────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	metricsServer := &http.Server{
		Addr:         cfg.MetricsAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() {
		logger.Info("metrics server listening", "addr", cfg.MetricsAddr)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server error", "error", err)
		}
	}()

	// ── gRPC listener ─────────────────────────────────────────────────────────
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		logger.Error("listen tcp", "addr", cfg.GRPCAddr, "error", err)
		os.Exit(1)
	}
	logger.Info("gRPC server listening", "addr", cfg.GRPCAddr)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("grpc serve error", "error", err)
		}
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down audit-trail service")
	cancel() // stop kafka consumers

	grpcServer.GracefulStop()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	_ = metricsServer.Shutdown(shutCtx)

	logger.Info("audit-trail service stopped")
}
