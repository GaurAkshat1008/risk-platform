package client

import (
	"context"
	"fmt"

	ingestion "graphql-bff/api/gen/ingestion"

	"google.golang.org/grpc"
)

type IngestionClient struct {
	stub ingestion.IngestionServiceClient
}

func NewIngestionClient(conn *grpc.ClientConn) *IngestionClient {
	return &IngestionClient{stub: ingestion.NewIngestionServiceClient(conn)}
}

func (c *IngestionClient) IngestPayment(ctx context.Context, req *ingestion.IngestPaymentRequest) (*ingestion.IngestionResult, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.IngestPayment(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ingestion.IngestPayment: %w", err)
	}
	return resp.Result, nil
}

func (c *IngestionClient) GetPaymentStatus(ctx context.Context, tenantID, idempotencyKey string) (*ingestion.GetPaymentStatusResponse, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.GetPaymentStatus(ctx, &ingestion.GetPaymentStatusRequest{
		TenantId:       tenantID,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, fmt.Errorf("ingestion.GetPaymentStatus: %w", err)
	}
	return resp, nil
}
