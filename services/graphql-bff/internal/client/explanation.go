package client

import (
	"context"
	"fmt"

	explanationpb "graphql-bff/api/gen/explanation"

	"google.golang.org/grpc"
)

type ExplanationClient struct {
	stub explanationpb.ExplanationServiceClient
}

func NewExplanationClient(conn *grpc.ClientConn) *ExplanationClient {
	return &ExplanationClient{stub: explanationpb.NewExplanationServiceClient(conn)}
}

func (c *ExplanationClient) GetExplanation(ctx context.Context, tenantID, paymentEventID string) (*explanationpb.Explanation, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.GetExplanation(ctx, &explanationpb.GetExplanationRequest{
		TenantId:       tenantID,
		PaymentEventId: paymentEventID,
	})
	if err != nil {
		return nil, fmt.Errorf("explanation.GetExplanation: %w", err)
	}
	return resp.Explanation, nil
}
