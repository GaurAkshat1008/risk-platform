package grpc

import (
	"context"
	"errors"
	"log/slog"

	pb "decision/api/gen/decision"
	"decision/internal/db"
	"decision/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DecisionService implements the gRPC DecisionService server.
type DecisionService struct {
	pb.UnimplementedDecisionServiceServer
	store   *db.DecisionStore
	metrics *telemetry.Metrics
	logger  *slog.Logger
}

func NewDecisionService(
	store *db.DecisionStore,
	metrics *telemetry.Metrics,
	logger *slog.Logger,
) *DecisionService {
	return &DecisionService{store: store, metrics: metrics, logger: logger}
}

// ── GetDecision ───────────────────────────────────────────────────────────────

func (s *DecisionService) GetDecision(
	ctx context.Context,
	req *pb.GetDecisionRequest,
) (*pb.GetDecisionResponse, error) {
	if req.PaymentEventId == "" {
		return nil, status.Error(codes.InvalidArgument, "payment_event_id is required")
	}
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	d, err := s.store.GetDecision(ctx, req.PaymentEventId, req.TenantId)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "decision not found for payment_event_id=%s tenant_id=%s",
				req.PaymentEventId, req.TenantId)
		}
		s.logger.Error("GetDecision store error", "error", err)
		return nil, status.Errorf(codes.Internal, "get decision: %v", err)
	}

	return &pb.GetDecisionResponse{Decision: toProto(d)}, nil
}

// ── OverrideDecision ──────────────────────────────────────────────────────────

func (s *DecisionService) OverrideDecision(
	ctx context.Context,
	req *pb.OverrideDecisionRequest,
) (*pb.OverrideDecisionResponse, error) {
	if req.DecisionId == "" {
		return nil, status.Error(codes.InvalidArgument, "decision_id is required")
	}
	if req.AnalystId == "" {
		return nil, status.Error(codes.InvalidArgument, "analyst_id is required")
	}
	if req.NewOutcome == pb.Outcome_OUTCOME_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "new_outcome is required")
	}
	if req.Reason == "" {
		return nil, status.Error(codes.InvalidArgument, "reason is required")
	}

	newOutcome := outcomeStr(req.NewOutcome)

	override, err := s.store.OverrideDecision(ctx, req.DecisionId, req.AnalystId, newOutcome, req.Reason)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "decision %s not found", req.DecisionId)
		}
		s.logger.Error("OverrideDecision store error", "error", err)
		return nil, status.Errorf(codes.Internal, "override decision: %v", err)
	}

	s.metrics.OverridesTotal.Add(ctx, 1,
		metric.WithAttributes(attribute.String("new_outcome", newOutcome)))

	s.logger.Info("decision overridden",
		"decision_id", req.DecisionId,
		"analyst_id", req.AnalystId,
		"new_outcome", newOutcome,
		"override_id", override.ID)

	return &pb.OverrideDecisionResponse{OverrideId: override.ID}, nil
}

// ── ListDecisions ─────────────────────────────────────────────────────────────

func (s *DecisionService) ListDecisions(
	ctx context.Context,
	req *pb.ListDecisionsRequest,
) (*pb.ListDecisionsResponse, error) {
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	page := int(req.Page)

	decisions, total, err := s.store.ListDecisions(ctx, req.TenantId, page, pageSize, req.OutcomeFilter)
	if err != nil {
		s.logger.Error("ListDecisions store error", "tenant_id", req.TenantId, "error", err)
		return nil, status.Errorf(codes.Internal, "list decisions: %v", err)
	}

	pbDecisions := make([]*pb.Decision, 0, len(decisions))
	for i := range decisions {
		pbDecisions = append(pbDecisions, toProto(&decisions[i]))
	}

	return &pb.ListDecisionsResponse{
		Decisions: pbDecisions,
		Total:     int32(total),
	}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func toProto(d *db.Decision) *pb.Decision {
	return &pb.Decision{
		Id:              d.ID,
		PaymentEventId:  d.PaymentEventID,
		TenantId:        d.TenantID,
		Outcome:         outcomeProto(d.Outcome),
		ReasonCodes:     d.ReasonCodes,
		ConfidenceScore: d.ConfidenceScore,
		Overridden:      d.Overridden,
		LatencyMs:       d.LatencyMs,
		CreatedAt:       timestamppb.New(d.CreatedAt),
	}
}

func outcomeStr(o pb.Outcome) string {
	switch o {
	case pb.Outcome_OUTCOME_FLAG:
		return "flag"
	case pb.Outcome_OUTCOME_REVIEW:
		return "review"
	case pb.Outcome_OUTCOME_BLOCK:
		return "block"
	default:
		return "approve"
	}
}

func outcomeProto(s string) pb.Outcome {
	switch s {
	case "flag":
		return pb.Outcome_OUTCOME_FLAG
	case "review":
		return pb.Outcome_OUTCOME_REVIEW
	case "block":
		return pb.Outcome_OUTCOME_BLOCK
	default:
		return pb.Outcome_OUTCOME_APPROVE
	}
}
