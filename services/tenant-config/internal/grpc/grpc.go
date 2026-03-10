package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"tenant-config/internal/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type Server struct {
	cfg		*config.Config
	logger	*slog.Logger
	grpcSrv *grpc.Server
	listner net.Listener
}

func (s *Server) GRPCServer() *grpc.Server {
	return s.grpcSrv
}

func NewServer(cfg *config.Config, logger *slog.Logger) (*Server, error) {
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	grpcSrv := grpc.NewServer()

	healthSvc := health.NewServer()
	healthSvc.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcSrv, healthSvc)

	return &Server{
		cfg: cfg,
		logger: logger,
		grpcSrv: grpcSrv,
		listner: lis,
	}, nil
}

func (s *Server) Start() error {
	s.logger.Info("Starting gRPC server", "addr", s.cfg.GRPCAddr)
	if err := s.grpcSrv.Serve(s.listner); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}
	return nil
}

func (s *Server) Stop(_ context.Context) error {
	s.logger.Info("Stopping gRPC server")
	s.grpcSrv.GracefulStop()
	return nil
}