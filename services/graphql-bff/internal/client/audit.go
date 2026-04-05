package client

import (
	"context"
	"fmt"

	audittrailpb "graphql-bff/api/gen/audit-trail"

	"google.golang.org/grpc"
)

type AuditClient struct {
	stub audittrailpb.AuditTrailServiceClient
}

func NewAuditClient(conn *grpc.ClientConn) *AuditClient {
	return &AuditClient{stub: audittrailpb.NewAuditTrailServiceClient(conn)}
}

func (c *AuditClient) AppendEvent(ctx context.Context, req *audittrailpb.AppendAuditEventRequest) (*audittrailpb.AuditEvent, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.AppendAuditEvent(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("audit.AppendAuditEvent: %w", err)
	}
	return resp.Event, nil
}

func (c *AuditClient) QueryTrail(ctx context.Context, req *audittrailpb.QueryAuditTrailRequest) ([]*audittrailpb.AuditEvent, string, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.QueryAuditTrail(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("audit.QueryAuditTrail: %w", err)
	}
	return resp.Events, resp.NextPageToken, nil
}

func (c *AuditClient) VerifyChain(ctx context.Context, tenantID string, limit int32) (bool, int64, string, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.VerifyChainIntegrity(ctx, &audittrailpb.VerifyChainRequest{
		TenantId: tenantID,
		Limit:    limit,
	})
	if err != nil {
		return false, 0, "", fmt.Errorf("audit.VerifyChainIntegrity: %w", err)
	}
	return resp.Valid, resp.EventsChecked, resp.BrokenAtId, nil
}
