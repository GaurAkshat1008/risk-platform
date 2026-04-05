package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	pb "explanation/api/gen/explanation"
	"explanation/internal/client"
	"explanation/internal/db"
	"explanation/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ExplanationService implements the gRPC ExplanationService server.
type ExplanationService struct {
	pb.UnimplementedExplanationServiceServer
	store         *db.ExplanationStore
	decisionCli   *client.DecisionClient
	rulesCli      *client.RulesClient
	metrics       *telemetry.Metrics
	logger        *slog.Logger
}

func NewExplanationService(
	store *db.ExplanationStore,
	decisionCli *client.DecisionClient,
	rulesCli *client.RulesClient,
	metrics *telemetry.Metrics,
	logger *slog.Logger,
) *ExplanationService {
	return &ExplanationService{
		store:       store,
		decisionCli: decisionCli,
		rulesCli:    rulesCli,
		metrics:     metrics,
		logger:      logger,
	}
}

// ── GetExplanation ────────────────────────────────────────────────────────────

func (s *ExplanationService) GetExplanation(
	ctx context.Context,
	req *pb.GetExplanationRequest,
) (*pb.GetExplanationResponse, error) {
	if req.PaymentEventId == "" {
		return nil, status.Error(codes.InvalidArgument, "payment_event_id is required")
	}
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	// Try the DB cache first.
	cached, err := s.store.GetByPaymentEvent(ctx, req.PaymentEventId, req.TenantId)
	if err == nil {
		s.metrics.ExplanationCacheHitsTotal.Add(ctx, 1,
			metric.WithAttributes(attribute.String("tenant_id", req.TenantId)))
		s.logger.Info("explanation served from cache",
			"payment_event_id", req.PaymentEventId, "tenant_id", req.TenantId)
		return &pb.GetExplanationResponse{Explanation: explanationToProto(cached)}, nil
	}
	if !errors.Is(err, db.ErrNotFound) {
		s.logger.Error("explanation store error", "error", err)
		return nil, status.Errorf(codes.Internal, "store lookup: %v", err)
	}

	// Generate a fresh explanation from upstream services.
	expl, err := s.generate(ctx, req.PaymentEventId, req.TenantId)
	if err != nil {
		s.metrics.DecisionFetchErrorsTotal.Add(ctx, 1,
			metric.WithAttributes(attribute.String("tenant_id", req.TenantId)))
		s.logger.Error("explanation generation failed",
			"payment_event_id", req.PaymentEventId, "error", err)
		return nil, status.Errorf(codes.Internal, "generate explanation: %v", err)
	}

	// Persist for future reads (best-effort cache).
	if err := s.store.Upsert(ctx, expl); err != nil {
		s.logger.Warn("failed to cache explanation (best-effort)", "error", err)
	}

	s.metrics.ExplanationsGeneratedTotal.Add(ctx, 1,
		metric.WithAttributes(attribute.String("tenant_id", req.TenantId)))
	s.logger.Info("explanation generated",
		"payment_event_id", req.PaymentEventId, "tenant_id", req.TenantId)

	return &pb.GetExplanationResponse{Explanation: explanationToProto(expl)}, nil
}

// ── GetNarrativeExplanation ───────────────────────────────────────────────────

func (s *ExplanationService) GetNarrativeExplanation(
	ctx context.Context,
	req *pb.GetNarrativeExplanationRequest,
) (*pb.GetNarrativeExplanationResponse, error) {
	if req.PaymentEventId == "" {
		return nil, status.Error(codes.InvalidArgument, "payment_event_id is required")
	}
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	// Reuse the full explanation (cached or generated).
	fullResp, err := s.GetExplanation(ctx, &pb.GetExplanationRequest{
		PaymentEventId: req.PaymentEventId,
		TenantId:       req.TenantId,
	})
	if err != nil {
		return nil, err
	}

	expl := fullResp.Explanation
	return &pb.GetNarrativeExplanationResponse{
		PaymentEventId: expl.PaymentEventId,
		Narrative:      expl.Narrative,
		PolicyVersion:  expl.PolicyVersion,
		GeneratedAt:    expl.GeneratedAt,
	}, nil
}

// ── private helpers ───────────────────────────────────────────────────────────

// generate fetches a decision + rule list from upstream services and builds an Explanation.
func (s *ExplanationService) generate(ctx context.Context, paymentEventID, tenantID string) (*db.Explanation, error) {
	// Fetch the decision from the Decision Service (keyed by payment_event_id).
	decision, err := s.decisionCli.GetDecision(ctx, paymentEventID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("fetch decision for payment_event %s: %w", paymentEventID, err)
	}

	// Fetch the enabled rule list for the tenant to provide labels/metadata.
	rules, err := s.rulesCli.ListRules(ctx, tenantID)
	if err != nil {
		// Non-fatal: proceed without rule details if Rules Engine is unavailable.
		s.logger.Warn("failed to fetch rules for explanation (proceeding without)", "error", err)
	}

	// Build a rule-name index.
	ruleNames := make(map[string]string, len(rules))
	for _, r := range rules {
		ruleNames[r.Id] = r.Name
	}

	// Map decision reason_codes → rule contributions.
	contributions := make([]db.RuleContribution, 0, len(decision.ReasonCodes))
	for _, code := range decision.ReasonCodes {
		name := ruleNames[code]
		if name == "" {
			name = code
		}
		contributions = append(contributions, db.RuleContribution{
			RuleID:   code,
			RuleName: name,
			Matched:  true,
			Action:   decision.Outcome.String(),
			Reason:   fmt.Sprintf("Rule %q contributed to the %s decision.", name, decision.Outcome.String()),
		})
	}

	// Build feature values from the decision payload.
	features := []db.FeatureValue{
		{Name: "payment_event_id", Value: decision.PaymentEventId},
		{Name: "confidence_score", Value: fmt.Sprintf("%.4f", decision.ConfidenceScore)},
		{Name: "latency_ms", Value: fmt.Sprintf("%d", decision.LatencyMs)},
		{Name: "overridden", Value: fmt.Sprintf("%v", decision.Overridden)},
	}

	narrative := buildNarrative(decision.Outcome.String(), contributions)
	policyVersion := fmt.Sprintf("1.0.0+decision-%s", decision.Id[:8])

	return &db.Explanation{
		DecisionID:        decision.Id,
		TenantID:          tenantID,
		PaymentEventID:    decision.PaymentEventId,
		Outcome:           decision.Outcome.String(),
		ConfidenceScore:   decision.ConfidenceScore,
		RuleContributions: contributions,
		FeatureValues:     features,
		Narrative:         narrative,
		PolicyVersion:     policyVersion,
	}, nil
}

// buildNarrative produces a concise natural-language summary from the rule contributions.
func buildNarrative(outcome string, contributions []db.RuleContribution) string {
	if len(contributions) == 0 {
		return fmt.Sprintf(
			"The payment was %s. No individual rules were matched; the decision reflects the tenant's default policy.",
			strings.ToLower(outcome),
		)
	}

	names := make([]string, 0, len(contributions))
	for _, c := range contributions {
		names = append(names, fmt.Sprintf("%q", c.RuleName))
	}

	list := strings.Join(names, ", ")
	return fmt.Sprintf(
		"The payment was %s based on %d matched rule(s): %s. "+
			"Each matched rule was evaluated against the payment context and collectively led to the %s outcome.",
		strings.ToLower(outcome), len(contributions), list, strings.ToLower(outcome),
	)
}

// explanationToProto converts a db.Explanation to its proto representation.
func explanationToProto(e *db.Explanation) *pb.Explanation {
	rules := make([]*pb.RuleContribution, 0, len(e.RuleContributions))
	for _, r := range e.RuleContributions {
		rules = append(rules, &pb.RuleContribution{
			RuleId:   r.RuleID,
			RuleName: r.RuleName,
			Matched:  r.Matched,
			Action:   r.Action,
			Reason:   r.Reason,
		})
	}

	features := make([]*pb.FeatureValue, 0, len(e.FeatureValues))
	for _, f := range e.FeatureValues {
		features = append(features, &pb.FeatureValue{
			Name:  f.Name,
			Value: f.Value,
		})
	}

	return &pb.Explanation{
		Id:                e.ID,
		DecisionId:        e.DecisionID,
		TenantId:          e.TenantID,
		PaymentEventId:    e.PaymentEventID,
		Outcome:           outcomeFromString(e.Outcome),
		ConfidenceScore:   e.ConfidenceScore,
		RuleContributions: rules,
		FeatureValues:     features,
		Narrative:         e.Narrative,
		PolicyVersion:     e.PolicyVersion,
		GeneratedAt:       timestamppb.New(e.GeneratedAt),
	}
}

func outcomeFromString(s string) pb.ExplanationOutcome {
	switch strings.ToUpper(s) {
	case "OUTCOME_APPROVE", "APPROVE":
		return pb.ExplanationOutcome_EXPLANATION_OUTCOME_APPROVE
	case "OUTCOME_FLAG", "FLAG":
		return pb.ExplanationOutcome_EXPLANATION_OUTCOME_FLAG
	case "OUTCOME_REVIEW", "REVIEW":
		return pb.ExplanationOutcome_EXPLANATION_OUTCOME_REVIEW
	case "OUTCOME_BLOCK", "BLOCK":
		return pb.ExplanationOutcome_EXPLANATION_OUTCOME_BLOCK
	default:
		return pb.ExplanationOutcome_EXPLANATION_OUTCOME_UNSPECIFIED
	}
}
