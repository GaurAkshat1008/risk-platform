package client

import (
	"context"
	"fmt"

	rulesengine "explanation/api/gen/rules-engine"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RulesClient is a gRPC client for the Rules Engine Service.
type RulesClient struct {
	client rulesengine.RulesEngineServiceClient
	conn   *grpc.ClientConn
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
		client: rulesengine.NewRulesEngineServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *RulesClient) Close() {
	_ = c.conn.Close()
}

// ListRules fetches all enabled rules for a tenant.
func (c *RulesClient) ListRules(ctx context.Context, tenantID string) ([]*rulesengine.Rule, error) {
	resp, err := c.client.ListRules(ctx, &rulesengine.ListRulesRequest{
		TenantId:        tenantID,
		IncludeDisabled: false,
	})
	if err != nil {
		return nil, fmt.Errorf("ListRules rpc: %w", err)
	}
	return resp.Rules, nil
}
