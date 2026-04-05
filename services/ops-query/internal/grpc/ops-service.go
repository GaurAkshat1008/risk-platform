package grpc

import (
	"context"
	"log/slog"
	"time"

	"ops-query/internal/client"
	"ops-query/internal/telemetry"

	pb "ops-query/api/gen/ops-query"

	"go.opentelemetry.io/otel/codes"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// OpsService implements the OpsQueryService gRPC server.
type OpsService struct {
	pb.UnimplementedOpsQueryServiceServer
	logClient  *client.LogClient
	promClient *client.PrometheusClient
	jaeger     *client.JaegerClient
	metrics    *telemetry.Metrics
	logger     *slog.Logger
}

func NewOpsService(
	logClient *client.LogClient,
	promClient *client.PrometheusClient,
	jaeger *client.JaegerClient,
	metrics *telemetry.Metrics,
	logger *slog.Logger,
) *OpsService {
	return &OpsService{
		logClient:  logClient,
		promClient: promClient,
		jaeger:     jaeger,
		metrics:    metrics,
		logger:     logger,
	}
}

// QueryLogs proxies the request to the log-ingestion service.
func (s *OpsService) QueryLogs(ctx context.Context, req *pb.QueryLogsRequest) (*pb.QueryLogsResponse, error) {
	s.metrics.QueryLogsTotal.Add(ctx, 1)

	resp, err := s.logClient.QueryLogs(ctx, req)
	if err != nil {
		telemetry.SpanFromContext(ctx).SetStatus(codes.Error, err.Error())
		s.logger.Error("QueryLogs upstream error", "error", err)
		return nil, status.Errorf(grpccodes.Internal, "query logs: %v", err)
	}
	return resp, nil
}

// QueryTraces retrieves spans from Jaeger.
func (s *OpsService) QueryTraces(ctx context.Context, req *pb.QueryTracesRequest) (*pb.QueryTracesResponse, error) {
	s.metrics.QueryTracesTotal.Add(ctx, 1)

	var (
		spans []client.JaegerSpan
		err   error
	)

	if tid := req.GetTraceId(); tid != "" {
		spans, err = s.jaeger.GetTrace(ctx, tid)
	} else {
		from := time.Now().Add(-time.Hour)
		to := time.Now()
		if req.GetFromTime() != nil {
			from = req.GetFromTime().AsTime()
		}
		if req.GetToTime() != nil {
			to = req.GetToTime().AsTime()
		}
		limit := int(req.GetLimit())
		if limit <= 0 {
			limit = 20
		}
		spans, err = s.jaeger.SearchTraces(ctx, req.GetService(), from, to, limit)
	}

	if err != nil {
		telemetry.SpanFromContext(ctx).SetStatus(codes.Error, err.Error())
		s.logger.Error("QueryTraces upstream error", "error", err)
		return nil, status.Errorf(grpccodes.Internal, "query traces: %v", err)
	}

	pbSpans := make([]*pb.Span, len(spans))
	for i, sp := range spans {
		pbSpans[i] = &pb.Span{
			TraceId:      sp.TraceID,
			SpanId:       sp.SpanID,
			ParentSpanId: sp.ParentSpanID,
			Service:      sp.ServiceName,
			Operation:    sp.OperationName,
			DurationMs:   sp.DurationMs,
			Status:       sp.Status,
			StartTime:    timestamppb.New(sp.StartTime),
		}
	}

	return &pb.QueryTracesResponse{Spans: pbSpans}, nil
}

// GetSLOStatus queries Prometheus for error rate and latency percentiles.
func (s *OpsService) GetSLOStatus(ctx context.Context, req *pb.GetSLOStatusRequest) (*pb.GetSLOStatusResponse, error) {
	s.metrics.GetSLOStatusTotal.Add(ctx, 1)

	window := req.GetWindow()
	if window == "" {
		window = "1h"
	}

	sloMetrics, err := s.promClient.GetSLOMetrics(ctx, req.GetService(), window)
	if err != nil {
		telemetry.SpanFromContext(ctx).SetStatus(codes.Error, err.Error())
		s.logger.Error("GetSLOStatus prometheus error", "error", err)
		return nil, status.Errorf(grpccodes.Internal, "get slo status: %v", err)
	}

	return &pb.GetSLOStatusResponse{
		Status: &pb.SLOStatus{
			Service:       req.GetService(),
			ErrorRate:     sloMetrics.ErrorRate,
			P50LatencyMs:  sloMetrics.P50LatencyMs,
			P95LatencyMs:  sloMetrics.P95LatencyMs,
			P99LatencyMs:  sloMetrics.P99LatencyMs,
			Availability:  sloMetrics.Availability,
			Window:        window,
		},
	}, nil
}

// ListAlerts retrieves active alerts from Prometheus and applies optional filters.
func (s *OpsService) ListAlerts(ctx context.Context, req *pb.ListAlertsRequest) (*pb.ListAlertsResponse, error) {
	s.metrics.ListAlertsTotal.Add(ctx, 1)

	promAlerts, err := s.promClient.GetAlerts(ctx)
	if err != nil {
		telemetry.SpanFromContext(ctx).SetStatus(codes.Error, err.Error())
		s.logger.Error("ListAlerts prometheus error", "error", err)
		return nil, status.Errorf(grpccodes.Internal, "list alerts: %v", err)
	}

	filterService := req.GetService()
	filterSeverity := req.GetSeverity()
	activeOnly := req.GetActiveOnly()
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 100
	}

	var alerts []*pb.Alert
	for _, a := range promAlerts {
		if activeOnly && a.State != "firing" {
			continue
		}
		if filterService != "" && a.Labels["service"] != filterService && a.Labels["job"] != filterService {
			continue
		}
		if filterSeverity != "" && a.Labels["severity"] != filterSeverity {
			continue
		}

		alert := &pb.Alert{
			Name:     a.Labels["alertname"],
			Service:  a.Labels["service"],
			Severity: a.Labels["severity"],
			State:    a.State,
			Summary:  a.Annotations["summary"],
			FiredAt:  timestamppb.New(a.ActiveAt),
		}
		alerts = append(alerts, alert)

		if len(alerts) >= limit {
			break
		}
	}

	return &pb.ListAlertsResponse{Alerts: alerts}, nil
}
