package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"tenant-config/internal/config"
	grpcserver "tenant-config/internal/grpc"

	"github.com/joho/godotenv"
)

func main() {
	_  = godotenv.Load(".env")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	slog.Info("starting tenant-config service", "config", cfg.String())

	src, err := grpcserver.NewServer(cfg, logger)
	if err != nil {
		slog.Error("Failed to create gRPC server", "error", err)
		os.Exit(1)
	}

	go func() {
		if err := src.Start(); err != nil {
			slog.Error("Failed to start gRPC server", "error",  err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down tenant-config service")
	src.Stop(context.Background())
	slog.Info("tenant-config service stopped")
}