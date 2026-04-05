package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	audittrailpb "audit-trail/api/gen/audit-trail"
	"audit-trail/internal/db"
	"audit-trail/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuditService implements the AuditTrailService gRPC server.
type AuditService struct {
	audittrailpb.UnimplementedAuditTrailServiceServer
	store   *db.AuditStore
	metrics *telemetry.Metrics
	logger  *slog.Logger
}

func NewAuditService(store *db.AuditStore, metrics *telemetry.Metrics, logger *slog.Logger) *AuditService {
	return &AuditService{
		store:   store,
		metrics: metrics,
		logger:  logger,
	}
}

// AppendAuditEvent appends a single event to the tamper-evident log.
func (s *AuditService) AppendAuditEvent(ctx context.Context, req *audittrailpb.AppendAuditEventRequest) (*audittrailpb.AppendAuditEventResponse, error) {
	span := telemetry.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("audit.tenant_id", req.GetTenantId()),
		attribute.String("audit.action", req.GetAction()),
		attribute.String("audit.resource_type", req.GetResourceType()),
	)

	if req.GetTenantId() == "" {
		return nil, status.Error(grpccodes.InvalidArgument, "tenant_id is required")
	}
	if req.GetAction() == "" {
		return nil, status.Error(grpccodes.InvalidArgument, "action is required")
	}

	params := db.AppendEventParams{
		TenantID:     req.GetTenantId(),
		ActorID:      req.GetActorId(),
		Action:       req.GetAction(),
		ResourceType: req.GetResourceType(),
		ResourceID:   req.GetResourceId(),
		SourceTopic:  req.GetSourceTopic(),
		Payload:      req.GetPayload(),
	}

	event, err := s.store.AppendEvent(ctx, params)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		s.logger.Error("append audit event failed", "error", err)
		return nil, status.Errorf(grpccodes.Internal, "append event: %v", err)
	}

	s.metrics.AuditEventsAppended.Add(ctx, 1)
	s.logger.Info("audit event appended", "id", event.ID, "seq", event.Seq, "tenant_id", event.TenantID)

	return &audittrailpb.AppendAuditEventResponse{
		Event: eventToProto(event),
	}, nil
}

// QueryAuditTrail returns a filtered, paginated list of audit events.
func (s *AuditService) QueryAuditTrail(ctx context.Context, req *audittrailpb.QueryAuditTrailRequest) (*audittrailpb.QueryAuditTrailResponse, error) {
	q := req.GetQuery()
	if q == nil {
		return nil, status.Error(grpccodes.InvalidArgument, "query is required")
	}
	if q.GetTenantId() == "" {
		return nil, status.Error(grpccodes.InvalidArgument, "tenant_id is required in query")
	}

	pageSize := int(q.GetPageSize())
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 500 {
		pageSize = 500
	}

	offset := 0
	if tok := q.GetPageToken(); tok != "" {
		off, err := strconv.Atoi(tok)
		if err == nil && off > 0 {
			offset = off
		}
	}

	params := db.AuditQueryParams{
		TenantID:     q.GetTenantId(),
		ActorID:      q.GetActorId(),
		ResourceType: q.GetResourceType(),
		ResourceID:   q.GetResourceId(),
		Action:       q.GetAction(),
		PageSize:     pageSize,
		Offset:       offset,
	}
	if q.GetFromTime() != nil {
		params.FromTime = q.GetFromTime().AsTime()
	}
	if q.GetToTime() != nil {
		params.ToTime = q.GetToTime().AsTime()
	}

	events, err := s.store.QueryEvents(ctx, params)
	if err != nil {
		span := telemetry.SpanFromContext(ctx)
		span.SetStatus(codes.Error, err.Error())
		return nil, status.Errorf(grpccodes.Internal, "query events: %v", err)
	}

	pbEvents := make([]*audittrailpb.AuditEvent, len(events))
	for i, e := range events {
		pbEvents[i] = eventToProto(&e)
	}

	var nextToken string
	if len(events) == pageSize {
		nextToken = fmt.Sprintf("%d", offset+pageSize)
	}

	return &audittrailpb.QueryAuditTrailResponse{
		Events:        pbEvents,
		NextPageToken: nextToken,
	}, nil
}

// VerifyChainIntegrity checks the hash chain for a tenant up to limit events.
func (s *AuditService) VerifyChainIntegrity(ctx context.Context, req *audittrailpb.VerifyChainRequest) (*audittrailpb.VerifyChainResponse, error) {
	if req.GetTenantId() == "" {
		return nil, status.Error(grpccodes.InvalidArgument, "tenant_id is required")
	}

	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 1000
	}

	valid, checked, brokenAtID, err := s.store.VerifyChain(ctx, req.GetTenantId(), limit)
	if err != nil {
		span := telemetry.SpanFromContext(ctx)
		span.SetStatus(codes.Error, err.Error())
		return nil, status.Errorf(grpccodes.Internal, "verify chain: %v", err)
	}

	s.metrics.ChainVerificationsTotal.Add(ctx, 1)
	s.logger.Info("chain verification complete",
		"tenant_id", req.GetTenantId(),
		"valid", valid,
		"events_checked", checked,
	)

	return &audittrailpb.VerifyChainResponse{
		Valid:         valid,
		EventsChecked: checked,
		BrokenAtId:    brokenAtID,
	}, nil
}

// eventToProto converts a db.AuditEvent to its protobuf representation.
func eventToProto(e *db.AuditEvent) *audittrailpb.AuditEvent {
	return &audittrailpb.AuditEvent{
		Id:           e.ID,
		TenantId:     e.TenantID,
		ActorId:      e.ActorID,
		Action:       e.Action,
		ResourceType: e.ResourceType,
		ResourceId:   e.ResourceID,
		SourceTopic:  e.SourceTopic,
		Payload:      e.Payload,
		PreviousHash: e.PreviousHash,
		Hash:         e.Hash,
		OccurredAt:   timestamppb.New(e.OccurredAt),
	}
}
