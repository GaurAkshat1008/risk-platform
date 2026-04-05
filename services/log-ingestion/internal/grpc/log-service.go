package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	pb "log-ingestion/api/gen/log-ingestion"
	"log-ingestion/internal/db"
	"log-ingestion/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// LogService implements the LogIngestionService gRPC server.
type LogService struct {
	pb.UnimplementedLogIngestionServiceServer
	store   *db.LogStore
	metrics *telemetry.Metrics
	logger  *slog.Logger
}

func NewLogService(store *db.LogStore, metrics *telemetry.Metrics, logger *slog.Logger) *LogService {
	return &LogService{store: store, metrics: metrics, logger: logger}
}

// IngestLog stores a single log entry submitted over gRPC.
func (s *LogService) IngestLog(ctx context.Context, req *pb.IngestLogRequest) (*pb.IngestLogResponse, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("log.service", req.GetService()),
		attribute.String("log.severity", req.GetSeverity()),
	)

	if req.GetService() == "" {
		return nil, status.Error(grpccodes.InvalidArgument, "service is required")
	}

	ts := time.Now().UTC()
	if req.GetTimestamp() != nil {
		ts = req.GetTimestamp().AsTime()
	}

	params := db.InsertParams{
		Service:     req.GetService(),
		Severity:    normaliseSeverity(req.GetSeverity()),
		Message:     req.GetMessage(),
		TraceID:     req.GetTraceId(),
		SpanID:      req.GetSpanId(),
		TenantID:    req.GetTenantId(),
		Environment: req.GetEnvironment(),
		Attributes:  req.GetAttributes(),
		Timestamp:   ts,
	}

	entry, err := s.store.Insert(ctx, params)
	if err != nil {
		trace.SpanFromContext(ctx).SetStatus(codes.Error, err.Error())
		s.logger.Error("ingest log failed", "error", err)
		return nil, status.Errorf(grpccodes.Internal, "ingest log: %v", err)
	}

	s.metrics.LogsIngestedTotal.Add(ctx, 1)
	s.logger.Debug("log ingested", "id", entry.ID, "service", entry.Service, "severity", entry.Severity)

	return &pb.IngestLogResponse{Id: entry.ID}, nil
}

// QueryLogs returns a filtered, paginated list of log entries.
func (s *LogService) QueryLogs(ctx context.Context, req *pb.QueryLogsRequest) (*pb.QueryLogsResponse, error) {
	q := req.GetQuery()
	if q == nil {
		return nil, status.Error(grpccodes.InvalidArgument, "query is required")
	}

	pageSize := int(q.GetPageSize())
	if pageSize <= 0 {
		pageSize = 50
	}

	offset := 0
	if tok := q.GetPageToken(); tok != "" {
		if off, err := strconv.Atoi(tok); err == nil && off > 0 {
			offset = off
		}
	}

	params := db.QueryParams{
		Service:         q.GetService(),
		Severity:        q.GetSeverity(),
		TraceID:         q.GetTraceId(),
		TenantID:        q.GetTenantId(),
		MessageContains: q.GetMessageContains(),
		PageSize:        pageSize,
		Offset:          offset,
	}
	if q.GetFromTime() != nil {
		params.FromTime = q.GetFromTime().AsTime()
	}
	if q.GetToTime() != nil {
		params.ToTime = q.GetToTime().AsTime()
	}

	entries, err := s.store.Query(ctx, params)
	if err != nil {
		trace.SpanFromContext(ctx).SetStatus(codes.Error, err.Error())
		return nil, status.Errorf(grpccodes.Internal, "query logs: %v", err)
	}

	s.metrics.QueryRequestsTotal.Add(ctx, 1)

	pbEntries := make([]*pb.LogEntry, len(entries))
	for i, e := range entries {
		pbEntries[i] = entryToProto(e)
	}

	var nextToken string
	if len(entries) == pageSize {
		nextToken = fmt.Sprintf("%d", offset+pageSize)
	}

	return &pb.QueryLogsResponse{
		Entries:       pbEntries,
		NextPageToken: nextToken,
	}, nil
}

func entryToProto(e *db.LogEntry) *pb.LogEntry {
	return &pb.LogEntry{
		Id:          e.ID,
		Service:     e.Service,
		Severity:    e.Severity,
		Message:     e.Message,
		TraceId:     e.TraceID,
		SpanId:      e.SpanID,
		TenantId:    e.TenantID,
		Environment: e.Environment,
		Attributes:  e.Attributes,
		Timestamp:   timestamppb.New(e.Timestamp),
	}
}

func normaliseSeverity(s string) string {
	switch strings.ToUpper(s) {
	case "DEBUG", "TRACE":
		return "DEBUG"
	case "INFO", "INFORMATION":
		return "INFO"
	case "WARN", "WARNING":
		return "WARN"
	case "ERROR", "ERR":
		return "ERROR"
	case "FATAL", "CRITICAL", "PANIC":
		return "FATAL"
	default:
		return "INFO"
	}
}
