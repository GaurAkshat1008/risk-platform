package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	pb "decision/api/gen/decision"
	"decision/internal/config"
	"decision/internal/db"
	grpcserver "decision/internal/grpc"
	"decision/internal/kafka"
	"decision/internal/telemetry"
	"decision/migrations"

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

	store := db.NewDecisionStore(pool)

	producer := kafka.NewProducer(kafka.Config{
		Brokers: strings.Split(cfg.Kafka.Brokers, ","),
		Topic:   cfg.Kafka.DecisionTopic,
	}, logger)
	defer producer.Close()
	publisher := kafka.NewDecisionEventPublisher(producer)
	slog.Info("kafka producer initialized")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start Kafka consumer in background
	consumer := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers:       cfg.Kafka.Brokers,
		RiskTopic:     cfg.Kafka.RiskTopic,
		ConsumerGroup: cfg.Kafka.ConsumerGroup,
	}, store, publisher, metrics, logger)
	defer consumer.Close()

	go consumer.Run(ctx)
	slog.Info("kafka consumer started", "topic", cfg.Kafka.RiskTopic)

	// Start gRPC server
	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(telemetry.UnaryServerInterceptor(metrics)),
	)

	decisionSvc := grpcserver.NewDecisionService(store, metrics, logger)
	pb.RegisterDecisionServiceServer(srv, decisionSvc)

	healthSvc := health.NewServer()
	healthSvc.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(srv, healthSvc)

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		slog.Info("metrics server listening", "addr", cfg.MetricsAddr)
		if err := http.ListenAndServe(cfg.MetricsAddr, nil); err != nil {
			slog.Error("metrics server failed", "error", err)
		}
	}()

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		slog.Error("failed to listen", "addr", cfg.GRPCAddr, "error", err)
		os.Exit(1)
	}

	go func() {
		slog.Info("gRPC server listening", "addr", cfg.GRPCAddr)
		if err := srv.Serve(lis); err != nil {
			slog.Error("gRPC server failed", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down decision service")

	srv.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := tp.Shutdown(shutdownCtx); err != nil {
		slog.Warn("tracer shutdown error", "error", err)
	}

	slog.Info("decision service stopped")
}
