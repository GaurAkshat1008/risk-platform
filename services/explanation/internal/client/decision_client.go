package client

import (
	"context"
	"fmt"

	decisionpb "explanation/api/gen/decision"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DecisionClient is a gRPC client for the Decision Service.
type DecisionClient struct {
	client decisionpb.DecisionServiceClient
	conn   *grpc.ClientConn
}

func NewDecisionClient(addr string) (*DecisionClient, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial decision-service %s: %w", addr, err)
	}
	return &DecisionClient{
		client: decisionpb.NewDecisionServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *DecisionClient) Close() {
	_ = c.conn.Close()
}

// GetDecision fetches a decision by payment_event_id and tenant_id.
func (c *DecisionClient) GetDecision(ctx context.Context, paymentEventID, tenantID string) (*decisionpb.Decision, error) {
	resp, err := c.client.GetDecision(ctx, &decisionpb.GetDecisionRequest{
		PaymentEventId: paymentEventID,
		TenantId:       tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("GetDecision rpc: %w", err)
	}
	return resp.Decision, nil
}
