package grpc

import (
	"context"
	"log/slog"

	pb "risk-orchestrator/api/gen/risk-orchestrator"
	"risk-orchestrator/internal/orchestrator"
	"risk-orchestrator/internal/telemetry"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RiskOrchestratorService implements the gRPC RiskOrchestratorService server.
type RiskOrchestratorService struct {
	pb.UnimplementedRiskOrchestratorServiceServer
	orch    *orchestrator.Orchestrator
	metrics *telemetry.Metrics
	logger  *slog.Logger
}

func NewRiskOrchestratorService(
	orch *orchestrator.Orchestrator,
	metrics *telemetry.Metrics,
	logger *slog.Logger,
) *RiskOrchestratorService {
	return &RiskOrchestratorService{orch: orch, metrics: metrics, logger: logger}
}

func (s *RiskOrchestratorService) EvaluateRisk(
	ctx context.Context,
	req *pb.RiskEvaluationRequest,
) (*pb.RiskEvaluationResponse, error) {
	if req.PaymentEventId == "" {
		return nil, status.Error(codes.InvalidArgument, "payment_event_id is required")
	}
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}

	result, err := s.orch.Evaluate(ctx, orchestrator.PaymentEvent{
		PaymentEventID: req.PaymentEventId,
		TenantID:       req.TenantId,
		Amount:         int64(req.Amount),
		Currency:       req.Currency,
		Source:         req.Source,
		Destination:    req.Destination,
		Metadata:       metadata,
	})
	if err != nil {
		s.logger.Error("EvaluateRisk failed",
			"payment_event_id", req.PaymentEventId,
			"tenant_id", req.TenantId,
			"error", err)
		return nil, status.Errorf(codes.Internal, "evaluation failed: %v", err)
	}

	pbResults := make([]*pb.RuleResultSummary, 0, len(result.RuleResults))
	for _, r := range result.RuleResults {
		pbResults = append(pbResults, &pb.RuleResultSummary{
			RuleId:   r.RuleID,
			RuleName: r.RuleName,
			Matched:  r.Matched,
			Action:   r.Action,
			Reason:   r.Reason,
		})
	}

	return &pb.RiskEvaluationResponse{
		PaymentEventId:  result.PaymentEventID,
		TenantId:        result.TenantID,
		AggregateAction: result.AggregateAction,
		RuleResults:     pbResults,
		LatencyMs:       result.LatencyMs,
		FailOpen:        result.FailOpen,
		EvaluatedAt:     timestamppb.Now(),
	}, nil
}
