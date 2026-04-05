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

    pb "ingestion/api/gen/ingestion"
    "ingestion/internal/cache"
    "ingestion/internal/config"
    "ingestion/internal/db"
    grpcserver "ingestion/internal/grpc"
    "ingestion/internal/kafka"
    "ingestion/internal/telemetry"
    "ingestion/migrations"

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

    store := db.NewPaymentStore(pool)

    dedup, err := cache.NewDedupeCache(cfg.Redis.Addr)
    if err != nil {
        slog.Error("failed to connect to redis", "error", err)
        os.Exit(1)
    }
    defer dedup.Close()
    slog.Info("connected to redis")

    producer := kafka.NewProducer(kafka.Config{
        Brokers: strings.Split(cfg.Kafka.Brokers, ","),
        Topic:   cfg.Kafka.PaymentsTopic,
    }, logger)
    defer producer.Close()
    publisher := kafka.NewPaymentEventPublisher(producer)
    slog.Info("kafka producer initialized")

    srv := grpc.NewServer(
        grpc.StatsHandler(otelgrpc.NewServerHandler()),
        grpc.UnaryInterceptor(telemetry.UnaryServerInterceptor(metrics)),
    )

    ingestionSvc := grpcserver.NewIngestionService(store, dedup, publisher, logger)
    pb.RegisterIngestionServiceServer(srv, ingestionSvc)

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
        slog.Info("ingestion gRPC server started", "addr", cfg.GRPCAddr)
        if err := srv.Serve(lis); err != nil {
            slog.Error("grpc serve failed", "error", err)
            os.Exit(1)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    slog.Info("shutting down ingestion")
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer shutdownCancel()

    srv.GracefulStop()
    _ = tp.Shutdown(shutdownCtx)
    slog.Info("ingestion stopped")
}