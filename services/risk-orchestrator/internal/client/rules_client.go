package client

import (
	"context"
	"fmt"

	pb "risk-orchestrator/api/gen/rules-engine"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RulesClient is a gRPC client for the Rules Engine service.
type RulesClient struct {
	client pb.RulesEngineServiceClient
	conn   *grpc.ClientConn
}

// EvalRequest carries the minimum context needed to call EvaluateRules.
type EvalRequest struct {
	PaymentEventID string
	TenantID       string
	Amount         int64
	Currency       string
	Source         string
	Destination    string
	Metadata       map[string]string
}

func NewRulesClient(addr string) (*RulesClient, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial rules-engine %s: %w", addr, err)
	}
	return &RulesClient{
		client: pb.NewRulesEngineServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *RulesClient) EvaluateRules(ctx context.Context, req EvalRequest) (*pb.EvaluateRulesResponse, error) {
	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}
	return c.client.EvaluateRules(ctx, &pb.EvaluateRulesRequest{
		Context: &pb.PaymentContext{
			PaymentEventId: req.PaymentEventID,
			TenantId:       req.TenantID,
			Amount:         req.Amount,
			Currency:       req.Currency,
			Source:         req.Source,
			Destination:    req.Destination,
			Metadata:       metadata,
		},
	})
}

func (c *RulesClient) Close() error {
	return c.conn.Close()
}
