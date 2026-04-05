package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "log-ingestion/api/gen/log-ingestion"
	"log-ingestion/internal/config"
	"log-ingestion/internal/db"
	grpcsvc "log-ingestion/internal/grpc"
	"log-ingestion/internal/kafka"
	"log-ingestion/internal/telemetry"
	"log-ingestion/migrations"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Tracing
	tp, err := telemetry.NewTracerProvider(ctx, telemetry.TracerConfig{
		OTLPEndpoint: cfg.OTel.CollectorEndpoint,
		ServiceName:  cfg.Service.Name,
		Environment:  cfg.Service.Env,
	})
	if err != nil {
		logger.Error("tracer provider", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		_ = tp.Shutdown(shutCtx)
	}()

	// Metrics
	mp, err := telemetry.NewMeterProvider()
	if err != nil {
		logger.Error("meter provider", "error", err)
		os.Exit(1)
	}
	metrics, err := telemetry.NewMetrics(mp)
	if err != nil {
		logger.Error("metrics init", "error", err)
		os.Exit(1)
	}

	// Database
	pool, err := db.NewPool(ctx, cfg.Postgres.DSN)
	if err != nil {
		logger.Error("db connect", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Migrations
	if err := migrations.Run(ctx, pool); err != nil {
		logger.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	store := db.NewLogStore(pool)

	// Kafka consumer (starts its own internal goroutine)
	consumer := kafka.NewConsumer(cfg.Kafka, store, logger)
	consumer.Start(ctx)

	// gRPC server
	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(telemetry.UnaryServerInterceptor(metrics)),
	)

	logSvc := grpcsvc.NewLogService(store, metrics, logger)
	pb.RegisterLogIngestionServiceServer(grpcSrv, logSvc)

	healthSvc := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSvc)
	healthSvc.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcSrv)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		logger.Error("tcp listen", "error", err)
		os.Exit(1)
	}

	// Metrics HTTP server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	})
	metricsSrv := &http.Server{Addr: cfg.MetricsAddr, Handler: mux}

	go func() {
		logger.Info("metrics server listening", "addr", cfg.MetricsAddr)
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server", "error", err)
		}
	}()

	go func() {
		logger.Info("gRPC server listening", "addr", cfg.GRPCAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			logger.Error("gRPC server stopped", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down log-ingestion service")

	consumer.Close()
	grpcSrv.GracefulStop()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = metricsSrv.Shutdown(shutCtx)

	logger.Info("shutdown complete")
}
