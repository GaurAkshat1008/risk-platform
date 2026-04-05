package grpc

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	pb "case-management/api/gen/case-management"
	"case-management/internal/db"
	"case-management/internal/kafka"
	"case-management/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CaseService implements the gRPC CaseManagementService server.
type CaseService struct {
	pb.UnimplementedCaseManagementServiceServer
	store     *db.CaseStore
	publisher *kafka.CaseEventPublisher
	metrics   *telemetry.Metrics
	logger    *slog.Logger
}

func NewCaseService(
	store *db.CaseStore,
	publisher *kafka.CaseEventPublisher,
	metrics *telemetry.Metrics,
	logger *slog.Logger,
) *CaseService {
	return &CaseService{
		store:     store,
		publisher: publisher,
		metrics:   metrics,
		logger:    logger,
	}
}

// ── CreateCase ────────────────────────────────────────────────────────────────

func (s *CaseService) CreateCase(
	ctx context.Context,
	req *pb.CreateCaseRequest,
) (*pb.CreateCaseResponse, error) {
	if req.DecisionId == "" {
		return nil, status.Error(codes.InvalidArgument, "decision_id is required")
	}
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	priority := priorityStr(req.Priority)
	slaDeadline := priorityToSLADeadline(priority)

	c, err := s.store.CreateCase(
		ctx,
		req.DecisionId,
		req.TenantId,
		req.PaymentEventId,
		req.Outcome,
		priority,
		slaDeadline,
	)
	if err != nil {
		s.logger.Error("CreateCase store error", "error", err)
		return nil, status.Errorf(codes.Internal, "create case: %v", err)
	}

	s.metrics.CasesCreatedTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("outcome", req.Outcome),
			attribute.String("priority", priority),
		))

	if err := s.publisher.PublishCaseCreated(ctx, c); err != nil {
		s.logger.Warn("publish case.created failed (best-effort)", "error", err)
	}

	return &pb.CreateCaseResponse{Case: caseToProto(c)}, nil
}

// ── GetCase ───────────────────────────────────────────────────────────────────

func (s *CaseService) GetCase(
	ctx context.Context,
	req *pb.GetCaseRequest,
) (*pb.GetCaseResponse, error) {
	if req.CaseId == "" {
		return nil, status.Error(codes.InvalidArgument, "case_id is required")
	}

	c, err := s.store.GetCase(ctx, req.CaseId)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "case %s not found", req.CaseId)
		}
		s.logger.Error("GetCase store error", "case_id", req.CaseId, "error", err)
		return nil, status.Errorf(codes.Internal, "get case: %v", err)
	}

	return &pb.GetCaseResponse{Case: caseToProto(c)}, nil
}

// ── AssignCase ────────────────────────────────────────────────────────────────

func (s *CaseService) AssignCase(
	ctx context.Context,
	req *pb.AssignCaseRequest,
) (*pb.AssignCaseResponse, error) {
	if req.CaseId == "" {
		return nil, status.Error(codes.InvalidArgument, "case_id is required")
	}
	if req.AssigneeId == "" {
		return nil, status.Error(codes.InvalidArgument, "assignee_id is required")
	}

	c, err := s.store.AssignCase(ctx, req.CaseId, req.AssigneeId, req.ActorId, "")
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "case %s not found", req.CaseId)
		}
		s.logger.Error("AssignCase store error", "case_id", req.CaseId, "error", err)
		return nil, status.Errorf(codes.Internal, "assign case: %v", err)
	}

	s.metrics.CasesAssignedTotal.Add(ctx, 1,
		metric.WithAttributes(attribute.String("assignee_id", req.AssigneeId)))

	s.logger.Info("case assigned",
		"case_id", c.ID, "assignee_id", req.AssigneeId, "actor_id", req.ActorId)

	return &pb.AssignCaseResponse{Case: caseToProto(c)}, nil
}

// ── UpdateCaseStatus ──────────────────────────────────────────────────────────

func (s *CaseService) UpdateCaseStatus(
	ctx context.Context,
	req *pb.UpdateCaseStatusRequest,
) (*pb.UpdateCaseStatusResponse, error) {
	if req.CaseId == "" {
		return nil, status.Error(codes.InvalidArgument, "case_id is required")
	}
	if req.Status == pb.CaseStatus_CASE_STATUS_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "status is required")
	}

	newStatus := statusStr(req.Status)

	c, err := s.store.UpdateCaseStatus(ctx, req.CaseId, newStatus, req.ActorId, req.Notes)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "case %s not found", req.CaseId)
		}
		s.logger.Error("UpdateCaseStatus store error", "case_id", req.CaseId, "error", err)
		return nil, status.Errorf(codes.Internal, "update case status: %v", err)
	}

	if newStatus == "resolved" {
		s.metrics.CasesResolvedTotal.Add(ctx, 1)
		if err := s.publisher.PublishCaseResolved(ctx, c, req.ActorId); err != nil {
			s.logger.Warn("publish case.resolved failed (best-effort)", "error", err)
		}
	}

	s.logger.Info("case status updated",
		"case_id", c.ID, "new_status", newStatus, "actor_id", req.ActorId)

	return &pb.UpdateCaseStatusResponse{Case: caseToProto(c)}, nil
}

// ── EscalateCase ──────────────────────────────────────────────────────────────

func (s *CaseService) EscalateCase(
	ctx context.Context,
	req *pb.EscalateCaseRequest,
) (*pb.EscalateCaseResponse, error) {
	if req.CaseId == "" {
		return nil, status.Error(codes.InvalidArgument, "case_id is required")
	}

	c, err := s.store.EscalateCase(ctx, req.CaseId, req.ActorId, req.Reason)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "case %s not found", req.CaseId)
		}
		s.logger.Error("EscalateCase store error", "case_id", req.CaseId, "error", err)
		return nil, status.Errorf(codes.Internal, "escalate case: %v", err)
	}

	s.metrics.CasesEscalatedTotal.Add(ctx, 1)

	if err := s.publisher.PublishCaseEscalated(ctx, c, req.Reason); err != nil {
		s.logger.Warn("publish case.escalated failed (best-effort)", "error", err)
	}

	s.logger.Info("case escalated",
		"case_id", c.ID, "actor_id", req.ActorId, "reason", req.Reason)

	return &pb.EscalateCaseResponse{Case: caseToProto(c)}, nil
}

// ── ListCases ─────────────────────────────────────────────────────────────────

func (s *CaseService) ListCases(
	ctx context.Context,
	req *pb.ListCasesRequest,
) (*pb.ListCasesResponse, error) {
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}

	offset := 0
	if req.PageToken != "" {
		if n, err := strconv.Atoi(req.PageToken); err == nil {
			offset = n
		}
	}

	statusFilter := statusStr(req.Status)
	if req.Status == pb.CaseStatus_CASE_STATUS_UNSPECIFIED {
		statusFilter = ""
	}

	cases, err := s.store.ListCases(ctx, req.TenantId, statusFilter, req.AssigneeId, pageSize, offset)
	if err != nil {
		s.logger.Error("ListCases store error", "tenant_id", req.TenantId, "error", err)
		return nil, status.Errorf(codes.Internal, "list cases: %v", err)
	}

	pbCases := make([]*pb.Case, 0, len(cases))
	for i := range cases {
		pbCases = append(pbCases, caseToProto(&cases[i]))
	}

	nextToken := ""
	if len(cases) == pageSize {
		nextToken = strconv.Itoa(offset + pageSize)
	}

	return &pb.ListCasesResponse{
		Cases:         pbCases,
		NextPageToken: nextToken,
	}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func caseToProto(c *db.Case) *pb.Case {
	return &pb.Case{
		Id:             c.ID,
		DecisionId:     c.DecisionID,
		TenantId:       c.TenantID,
		AssigneeId:     c.AssigneeID,
		Status:         protoStatus(c.Status),
		Priority:       protoPriority(c.Priority),
		PaymentEventId: c.PaymentEventID,
		Outcome:        c.Outcome,
		SlaDeadline:    timestamppb.New(c.SLADeadline),
		CreatedAt:      timestamppb.New(c.CreatedAt),
		UpdatedAt:      timestamppb.New(c.UpdatedAt),
	}
}

func statusStr(s pb.CaseStatus) string {
	switch s {
	case pb.CaseStatus_CASE_STATUS_OPEN:
		return "open"
	case pb.CaseStatus_CASE_STATUS_IN_REVIEW:
		return "in_review"
	case pb.CaseStatus_CASE_STATUS_RESOLVED:
		return "resolved"
	case pb.CaseStatus_CASE_STATUS_ESCALATED:
		return "escalated"
	default:
		return ""
	}
}

func protoStatus(s string) pb.CaseStatus {
	switch s {
	case "open":
		return pb.CaseStatus_CASE_STATUS_OPEN
	case "in_review":
		return pb.CaseStatus_CASE_STATUS_IN_REVIEW
	case "resolved":
		return pb.CaseStatus_CASE_STATUS_RESOLVED
	case "escalated":
		return pb.CaseStatus_CASE_STATUS_ESCALATED
	default:
		return pb.CaseStatus_CASE_STATUS_UNSPECIFIED
	}
}

func priorityStr(p pb.CasePriority) string {
	switch p {
	case pb.CasePriority_CASE_PRIORITY_LOW:
		return "low"
	case pb.CasePriority_CASE_PRIORITY_MEDIUM:
		return "medium"
	case pb.CasePriority_CASE_PRIORITY_HIGH:
		return "high"
	case pb.CasePriority_CASE_PRIORITY_CRITICAL:
		return "critical"
	default:
		return "medium"
	}
}

func protoPriority(p string) pb.CasePriority {
	switch p {
	case "low":
		return pb.CasePriority_CASE_PRIORITY_LOW
	case "medium":
		return pb.CasePriority_CASE_PRIORITY_MEDIUM
	case "high":
		return pb.CasePriority_CASE_PRIORITY_HIGH
	case "critical":
		return pb.CasePriority_CASE_PRIORITY_CRITICAL
	default:
		return pb.CasePriority_CASE_PRIORITY_UNSPECIFIED
	}
}

func priorityToSLADeadline(priority string) time.Time {
	now := time.Now().UTC()
	switch priority {
	case "critical":
		return now.Add(4 * time.Hour)
	case "high":
		return now.Add(24 * time.Hour)
	case "medium":
		return now.Add(48 * time.Hour)
	default:
		return now.Add(72 * time.Hour)
	}
}
