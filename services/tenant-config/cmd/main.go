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

    pb "tenant-config/api/gen/tenant"
    "tenant-config/internal/cache"
    "tenant-config/internal/config"
    "tenant-config/internal/db"
    grpcserver "tenant-config/internal/grpc"
    "tenant-config/internal/kafka"
    "tenant-config/internal/telemetry"

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

    // OTel tracing
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

    // Prometheus metrics
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

    // PostgreSQL
    pool, err := db.NewPool(initCtx, cfg.Postgres.DSN)
    if err != nil {
        slog.Error("failed to connect to postgres", "error", err)
        os.Exit(1)
    }
    defer pool.Close()
    slog.Info("connected to postgres")

    store := db.NewTenantStore(pool)

    // Redis
    tenantCache, err := cache.NewTenantCache(cfg.Redis.Addr)
    if err != nil {
        slog.Error("failed to connect to redis", "error", err)
        os.Exit(1)
    }
    defer tenantCache.Close()
    slog.Info("connected to redis")

    // Kafka
    kafkaProducer := kafka.NewProducer(kafka.Config{
        Brokers: strings.Split(cfg.Kafka.Brokers, ","),
        Topic:   cfg.Kafka.TenantTopic,
    }, logger)
    defer kafkaProducer.Close()
    eventPublisher := kafka.NewTenantEventPublisher(kafkaProducer)
    slog.Info("kafka producer initialized")

    // gRPC server
    srv := grpc.NewServer(
        grpc.StatsHandler(otelgrpc.NewServerHandler()),
        grpc.UnaryInterceptor(telemetry.UnaryServerInterceptor(metrics)),
    )

    tenantSvc := grpcserver.NewTenantConfigService(store, tenantCache, eventPublisher, logger)
    pb.RegisterTenantConfigServiceServer(srv, tenantSvc)

    healthSvc := health.NewServer()
    healthSvc.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
    healthpb.RegisterHealthServer(srv, healthSvc)

    // Prometheus HTTP endpoint
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
        slog.Info("tenant-config gRPC server started", "addr", cfg.GRPCAddr)
        if err := srv.Serve(lis); err != nil {
            slog.Error("grpc serve failed", "error", err)
            os.Exit(1)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    slog.Info("shutting down tenant-config")
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer shutdownCancel()

    srv.GracefulStop()
    _ = tp.Shutdown(shutdownCtx)
    slog.Info("tenant-config stopped")
}