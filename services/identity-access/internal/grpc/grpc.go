package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"identity-access/internal/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type Server struct {
	cfg config.Config
	logger *slog.Logger
	grpcSrv *grpc.Server
	listener net.Listener
}

func (s *Server) GRPCServer() *grpc.Server {
    return s.grpcSrv
}

func NewServer(cfg config.Config, logger *slog.Logger) (*Server, error) {
	lis, err := net.Listen("tcp", cfg.GRPCAddr)

	if err != nil {
		return nil, fmt.Errorf("Listen failed on %s: %w", cfg.GRPCAddr, err)
	}

	grpcSrv := grpc.NewServer() 

	healthSvc := health.NewServer()
	healthSvc.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcSrv, healthSvc)

	return &Server{
		cfg: cfg,
		logger: logger,
		grpcSrv: grpcSrv,
		listener: lis,
	}, nil
}

func (s *Server) Start() error {
	s.logger.Info("Starting gRPC server", "address", s.cfg.GRPCAddr)
	if err := s.grpcSrv.Serve(s.listener); err != nil {
		return fmt.Errorf("Failed to start gRPC server: %w", err)
	}
	return nil
}

func (s *Server) Stop(_ context.Context) error {
	s.logger.Info("Stopping gRPC server")
	s.grpcSrv.GracefulStop()
	return nil
}