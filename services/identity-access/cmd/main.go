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

	identitypb "identity-access/api/gen/identity"
	"identity-access/internal/auth"
	"identity-access/internal/config"
	grpcserver "identity-access/internal/grpc"
	"identity-access/internal/kafka"
	"identity-access/internal/keycloak"
	"identity-access/internal/rbac"
	"identity-access/internal/telemetry"

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
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger := slog.New((slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})))
	slog.SetDefault(logger)

	// Short context for telemetry init only
	initCx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer initCancel()

	// Longer background context for Keycloak (may be slow to start)
	keycloakCx, keycloakCancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer keycloakCancel()

	tp, err := telemetry.NewTracerProvider(initCx, telemetry.TracerConfig{
		ServiceName:       cfg.Service.Name,
		ServiceVersion:    "0.1.0",
		Environment:       cfg.Service.Env,
		CollectorEndpoint: cfg.OTel.CollectorEndpoint,
	})

	if err != nil {
		slog.Error("Failed to initialize telemetry", "error", err)
		os.Exit(1)
	}
	slog.Info("tracing initialized")

	mp, err := telemetry.NewMeterProvider()
	if err != nil {
		slog.Error("Failed to initialize metrics", "error", err)
		os.Exit(1)
	}
	metrics, err := telemetry.NewMetrics(mp)
	if err != nil {
		slog.Error("metrics init failed", "error", err)
		os.Exit(1)
	}
	slog.Info("metrics initialized")

	// Retry OIDC discovery until Keycloak is ready (up to 2 min)
	var validator *auth.Validator
	for attempt := 1; ; attempt++ {
		attemptCx, attemptCancel := context.WithTimeout(keycloakCx, 15*time.Second)
		validator, err = auth.NewValidator(attemptCx, cfg.Keycloak.Issuer, cfg.Keycloak.Audience)
		attemptCancel()
		if err == nil {
			break
		}
		slog.Warn("Keycloak not ready, retrying OIDC discovery", "attempt", attempt, "error", err)
		if keycloakCx.Err() != nil {
			slog.Error("Keycloak did not become ready in time", "error", err)
			os.Exit(1)
		}
		time.Sleep(5 * time.Second)
	}

	rbac := rbac.NewEvaluator()

	keycloakClient := keycloak.NewClient(keycloak.Config{
		BaseURL:      cfg.Keycloak.BaseURL,
		Realm:        cfg.Keycloak.Realm,
		ClientID:     cfg.Keycloak.ClientID,
		ClientSecret: cfg.Keycloak.ClientSecret,
	}, logger)

	if err := keycloakClient.Ping(keycloakCx); err != nil {
		slog.Error("Failed to connect to Keycloak", "error", err)
		os.Exit(1)
	}
	slog.Info("Successfully connected to Keycloak")

	kafkaProducer := kafka.NewProducer(kafka.Config{
		Brokers: strings.Split(cfg.Kafka.Brokers, ","),
		Topic:   cfg.Kafka.AuthEventsTopic,
	}, logger)

	authPublisher := kafka.NewAuthEventPublisher(kafkaProducer)
	slog.Info("Kafka producer initialized and connected")

	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(telemetry.UnaryServerInterceptor(metrics)),
	)

	identitySvc := grpcserver.NewIdentityServiceServer(validator,
		rbac,
		keycloakClient,
		authPublisher,
		metrics)
	identitypb.RegisterIdentityAccessServiceServer(srv, identitySvc)

	healthSvc := health.NewServer()
	healthSvc.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(srv, healthSvc)

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		slog.Info("Prometheus metrics available", "addr", cfg.MetricsAddr)
		if err := http.ListenAndServe(cfg.MetricsAddr, nil); err != nil {
			slog.Error("Metrics server failed", "error", err)
		}
	}()

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		slog.Error("Failed to listen on gRPC address", "error", err)
		os.Exit(1)
	}

	go func() {
		slog.Info("Identity-access started", "grpc_addr", cfg.GRPCAddr)
		if err := srv.Serve(lis); err != nil {
			slog.Error("grpc serve failed", "error", "err")
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down Identity Access Service")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	srv.GracefulStop()
	_ = kafkaProducer.Close()
	_ = tp.Shutdown(ctx)

	time.Sleep(2 * time.Second)

	slog.Info("Identity Access Service stopped gracefully")
}
