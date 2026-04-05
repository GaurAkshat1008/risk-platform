package client

import (
	"context"
	"fmt"

	pb "ops-query/api/gen/log-ingestion"
	oppb "ops-query/api/gen/ops-query"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// LogClient wraps the log-ingestion gRPC client.
type LogClient struct {
	conn   *grpc.ClientConn
	client pb.LogIngestionServiceClient
}

// NewLogClient creates a new gRPC client connected to the log-ingestion service.
func NewLogClient(addr string) (*LogClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial log-ingestion %s: %w", addr, err)
	}
	return &LogClient{conn: conn, client: pb.NewLogIngestionServiceClient(conn)}, nil
}

// Close releases the underlying connection.
func (c *LogClient) Close() error {
	return c.conn.Close()
}

// QueryLogs calls log-ingestion.QueryLogs and converts results to ops-query proto types.
func (c *LogClient) QueryLogs(ctx context.Context, req *oppb.QueryLogsRequest) (*oppb.QueryLogsResponse, error) {
	q := req.GetQuery()
	pbQuery := &pb.LogQuery{}
	if q != nil {
		pbQuery.Service = q.GetService()
		pbQuery.Severity = q.GetSeverity()
		pbQuery.TraceId = q.GetTraceId()
		pbQuery.TenantId = q.GetTenantId()
		pbQuery.MessageContains = q.GetMessageContains()
		pbQuery.PageSize = q.GetPageSize()
		pbQuery.PageToken = q.GetPageToken()
		pbQuery.FromTime = q.GetFromTime()
		pbQuery.ToTime = q.GetToTime()
	}

	resp, err := c.client.QueryLogs(ctx, &pb.QueryLogsRequest{Query: pbQuery})
	if err != nil {
		return nil, fmt.Errorf("log-ingestion QueryLogs: %w", err)
	}

	entries := make([]*oppb.LogEntry, len(resp.GetEntries()))
	for i, e := range resp.GetEntries() {
		entries[i] = &oppb.LogEntry{
			Id:          e.GetId(),
			Service:     e.GetService(),
			Severity:    e.GetSeverity(),
			Message:     e.GetMessage(),
			TraceId:     e.GetTraceId(),
			SpanId:      e.GetSpanId(),
			TenantId:    e.GetTenantId(),
			Environment: e.GetEnvironment(),
			Attributes:  e.GetAttributes(),
			Timestamp:   e.GetTimestamp(),
		}
	}

	return &oppb.QueryLogsResponse{
		Entries:       entries,
		NextPageToken: resp.GetNextPageToken(),
	}, nil
}
