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

	pb "ops-query/api/gen/ops-query"
	"ops-query/internal/client"
	"ops-query/internal/config"
	grpcsvc "ops-query/internal/grpc"
	"ops-query/internal/telemetry"

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

	// Downstream clients
	logClient, err := client.NewLogClient(cfg.LogIngestion.Addr)
	if err != nil {
		logger.Error("log-ingestion client", "error", err)
		os.Exit(1)
	}
	defer logClient.Close()

	promClient := client.NewPrometheusClient(cfg.Prometheus.Addr)
	jaegerClient := client.NewJaegerClient(cfg.Jaeger.Addr)

	// gRPC server
	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(telemetry.UnaryServerInterceptor(metrics)),
	)

	opsSvc := grpcsvc.NewOpsService(logClient, promClient, jaegerClient, metrics, logger)
	pb.RegisterOpsQueryServiceServer(grpcSrv, opsSvc)

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
	logger.Info("shutting down ops-query service")

	grpcSrv.GracefulStop()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = metricsSrv.Shutdown(shutCtx)

	logger.Info("shutdown complete")
}
