package client

import (
	"context"
	"fmt"

	decisionpb "graphql-bff/api/gen/decision"
	"graphql-bff/internal/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type DecisionClient struct {
	stub decisionpb.DecisionServiceClient
}

func NewDecisionClient(conn *grpc.ClientConn) *DecisionClient {
	return &DecisionClient{stub: decisionpb.NewDecisionServiceClient(conn)}
}

func (c *DecisionClient) GetDecision(ctx context.Context, tenantID, paymentEventID string) (*decisionpb.Decision, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.GetDecision(ctx, &decisionpb.GetDecisionRequest{
		TenantId:       tenantID,
		PaymentEventId: paymentEventID,
	})
	if err != nil {
		return nil, fmt.Errorf("decision.GetDecision: %w", err)
	}
	return resp.Decision, nil
}

func (c *DecisionClient) ListDecisions(ctx context.Context, tenantID string, page, pageSize int32, outcomeFilter string) ([]*decisionpb.Decision, int32, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.ListDecisions(ctx, &decisionpb.ListDecisionsRequest{
		TenantId:      tenantID,
		Page:          page,
		PageSize:      pageSize,
		OutcomeFilter: outcomeFilter,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("decision.ListDecisions: %w", err)
	}
	return resp.Decisions, resp.Total, nil
}

func (c *DecisionClient) OverrideDecision(ctx context.Context, decisionID, analystID string, newOutcome decisionpb.Outcome, reason string) (string, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.OverrideDecision(ctx, &decisionpb.OverrideDecisionRequest{
		DecisionId: decisionID,
		AnalystId:  analystID,
		NewOutcome: newOutcome,
		Reason:     reason,
	})
	if err != nil {
		return "", fmt.Errorf("decision.OverrideDecision: %w", err)
	}
	return resp.OverrideId, nil
}

// attachToken reads the Bearer token from context and adds it to outgoing gRPC metadata.
func attachToken(ctx context.Context) context.Context {
	if token := auth.TokenFromContext(ctx); token != "" {
		md := metadata.Pairs("authorization", "Bearer "+token)
		return metadata.NewOutgoingContext(ctx, md)
	}
	return ctx
}
