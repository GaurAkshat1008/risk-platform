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

	pb "notification/api/gen/notification"
	"notification/internal/cache"
	"notification/internal/config"
	"notification/internal/db"
	grpcserver "notification/internal/grpc"
	"notification/internal/kafka"
	"notification/internal/telemetry"
	"notification/migrations"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	_ = godotenv.Load(".env")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer initCancel()

	tp, err := telemetry.NewTracerProvider(initCtx, telemetry.TracerConfig{
		ServiceName:       cfg.Service.Name,
		ServiceVersion:    "0.1.0",
		Environment:       cfg.Service.Env,
		CollectorEndpoint: cfg.OTel.CollectorEndpoint,
	})
	if err != nil {
		slog.Error("failed to initialize tracer", "error", err)
		os.Exit(1)
	}
	defer tp.Shutdown(context.Background()) //nolint:errcheck
	slog.Info("tracing initialized")

	mp, err := telemetry.NewMeterProvider()
	if err != nil {
		slog.Error("failed to initialize meter provider", "error", err)
		os.Exit(1)
	}
	metrics, err := telemetry.NewMetrics(mp)
	if err != nil {
		slog.Error("failed to initialize metrics", "error", err)
		os.Exit(1)
	}
	slog.Info("metrics initialized")

	pool, err := db.NewPool(initCtx, cfg.Postgres.DSN)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("connected to postgres")

	if err := migrations.Run(initCtx, pool); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")

	store := db.NewNotificationStore(pool)

	// Rate limit: max 60 notifications per tenant per channel per minute.
	rateLimiter, err := cache.NewRateLimiter(cfg.Redis.Addr, 60)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rateLimiter.Close()
	slog.Info("connected to redis")

	// Kafka consumer — reads domain events from multiple topics.
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()

	consumer := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.Topics,
		cfg.Kafka.ConsumerGroup,
		store, rateLimiter, logger,
	)
	consumer.Start(consumerCtx)
	defer consumer.Close()
	slog.Info("kafka consumer started", "topics", cfg.Kafka.Topics)

	// gRPC server
	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(telemetry.UnaryServerInterceptor(metrics)),
	)

	notificationSvc := grpcserver.NewNotificationService(store, rateLimiter, metrics, logger)
	pb.RegisterNotificationServiceServer(srv, notificationSvc)

	healthSvc := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSvc)
	healthSvc.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		slog.Error("failed to listen", "addr", cfg.GRPCAddr, "error", err)
		os.Exit(1)
	}

	// Prometheus metrics endpoint
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsSrv := &http.Server{Addr: cfg.MetricsAddr, Handler: metricsMux}

	go func() {
		slog.Info("metrics server listening", "addr", cfg.MetricsAddr)
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server error", "error", err)
		}
	}()

	go func() {
		slog.Info("gRPC server listening", "addr", cfg.GRPCAddr)
		if err := srv.Serve(lis); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down notification service")

	consumerCancel()
	srv.GracefulStop()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	if err := metricsSrv.Shutdown(shutCtx); err != nil {
		slog.Warn("metrics server shutdown error", "error", err)
	}

	slog.Info("notification service stopped")
}
