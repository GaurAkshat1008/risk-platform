package client

import (
	"context"
	"fmt"

	opsquerypb "graphql-bff/api/gen/ops-query"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type OpsQueryClient struct {
	stub opsquerypb.OpsQueryServiceClient
}

func NewOpsQueryClient(conn *grpc.ClientConn) *OpsQueryClient {
	return &OpsQueryClient{stub: opsquerypb.NewOpsQueryServiceClient(conn)}
}

func (c *OpsQueryClient) QueryLogs(ctx context.Context, req *opsquerypb.QueryLogsRequest) ([]*opsquerypb.LogEntry, string, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.QueryLogs(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("ops.QueryLogs: %w", err)
	}
	return resp.Entries, resp.NextPageToken, nil
}

func (c *OpsQueryClient) QueryTraces(ctx context.Context, traceID, service string, from, to *time.Time, limit int32) ([]*opsquerypb.Span, error) {
	ctx = attachToken(ctx)
	req := &opsquerypb.QueryTracesRequest{
		TraceId: traceID,
		Service: service,
		Limit:   limit,
	}
	if from != nil {
		req.FromTime = timestamppb.New(*from)
	}
	if to != nil {
		req.ToTime = timestamppb.New(*to)
	}
	resp, err := c.stub.QueryTraces(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ops.QueryTraces: %w", err)
	}
	return resp.Spans, nil
}

func (c *OpsQueryClient) GetSLOStatus(ctx context.Context, service, window string) (*opsquerypb.SLOStatus, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.GetSLOStatus(ctx, &opsquerypb.GetSLOStatusRequest{
		Service: service,
		Window:  window,
	})
	if err != nil {
		return nil, fmt.Errorf("ops.GetSLOStatus: %w", err)
	}
	return resp.Status, nil
}

func (c *OpsQueryClient) ListAlerts(ctx context.Context, service, severity string, activeOnly bool, limit int32) ([]*opsquerypb.Alert, error) {
	ctx = attachToken(ctx)
	resp, err := c.stub.ListAlerts(ctx, &opsquerypb.ListAlertsRequest{
		Service:    service,
		Severity:   severity,
		ActiveOnly: activeOnly,
		Limit:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("ops.ListAlerts: %w", err)
	}
	return resp.Alerts, nil
}
