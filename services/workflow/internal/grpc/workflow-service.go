package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	pb "workflow/api/gen/workflow"
	"workflow/internal/cache"
	"workflow/internal/db"
	"workflow/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// WorkflowService implements the gRPC WorkflowService server.
type WorkflowService struct {
	pb.UnimplementedWorkflowServiceServer
	store   *db.WorkflowStore
	cache   *cache.WorkflowCache
	metrics *telemetry.Metrics
	logger  *slog.Logger
}

func NewWorkflowService(
	store *db.WorkflowStore,
	wfCache *cache.WorkflowCache,
	metrics *telemetry.Metrics,
	logger *slog.Logger,
) *WorkflowService {
	return &WorkflowService{
		store:   store,
		cache:   wfCache,
		metrics: metrics,
		logger:  logger,
	}
}

// ── CreateWorkflowTemplate ────────────────────────────────────────────────────

func (s *WorkflowService) CreateWorkflowTemplate(
	ctx context.Context,
	req *pb.CreateWorkflowTemplateRequest,
) (*pb.CreateWorkflowTemplateResponse, error) {
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if len(req.States) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one state is required")
	}

	transitions := protoToTransitions(req.Transitions)

	tmpl, err := s.store.CreateTemplate(ctx, req.TenantId, req.Name, req.States, transitions)
	if err != nil {
		if errors.Is(err, db.ErrAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "workflow template %q already exists", req.Name)
		}
		s.logger.Error("CreateWorkflowTemplate store error", "error", err)
		return nil, status.Errorf(codes.Internal, "create template: %v", err)
	}

	s.metrics.TemplatesCreatedTotal.Add(ctx, 1,
		metric.WithAttributes(attribute.String("tenant_id", req.TenantId)))

	if err := s.cache.Set(ctx, tmpl); err != nil {
		s.logger.Warn("failed to cache new template (best-effort)", "error", err)
	}

	s.logger.Info("workflow template created",
		"template_id", tmpl.ID,
		"tenant_id", tmpl.TenantID,
		"name", tmpl.Name,
	)
	return &pb.CreateWorkflowTemplateResponse{Template: templateToProto(tmpl)}, nil
}

// ── GetWorkflowTemplate ───────────────────────────────────────────────────────

func (s *WorkflowService) GetWorkflowTemplate(
	ctx context.Context,
	req *pb.GetWorkflowTemplateRequest,
) (*pb.GetWorkflowTemplateResponse, error) {
	if req.TemplateId == "" {
		return nil, status.Error(codes.InvalidArgument, "template_id is required")
	}
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	// Cache-aside read
	cached, err := s.cache.Get(ctx, req.TemplateId, req.TenantId)
	if err != nil {
		s.logger.Warn("cache get failed, falling back to DB", "error", err)
	}
	if cached != nil {
		s.metrics.CacheHitsTotal.Add(ctx, 1)
		return &pb.GetWorkflowTemplateResponse{Template: templateToProto(cached)}, nil
	}
	s.metrics.CacheMissesTotal.Add(ctx, 1)

	tmpl, err := s.store.GetTemplate(ctx, req.TemplateId, req.TenantId)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "template %s not found", req.TemplateId)
		}
		s.logger.Error("GetWorkflowTemplate store error", "template_id", req.TemplateId, "error", err)
		return nil, status.Errorf(codes.Internal, "get template: %v", err)
	}

	if err := s.cache.Set(ctx, tmpl); err != nil {
		s.logger.Warn("failed to warm cache (best-effort)", "error", err)
	}

	return &pb.GetWorkflowTemplateResponse{Template: templateToProto(tmpl)}, nil
}

// ── UpdateWorkflowTemplate ────────────────────────────────────────────────────

func (s *WorkflowService) UpdateWorkflowTemplate(
	ctx context.Context,
	req *pb.UpdateWorkflowTemplateRequest,
) (*pb.UpdateWorkflowTemplateResponse, error) {
	if req.TemplateId == "" {
		return nil, status.Error(codes.InvalidArgument, "template_id is required")
	}
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}
	if len(req.States) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one state is required")
	}

	transitions := protoToTransitions(req.Transitions)

	tmpl, err := s.store.UpdateTemplate(ctx, req.TemplateId, req.TenantId, req.States, transitions)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "template %s not found", req.TemplateId)
		}
		s.logger.Error("UpdateWorkflowTemplate store error", "template_id", req.TemplateId, "error", err)
		return nil, status.Errorf(codes.Internal, "update template: %v", err)
	}

	s.metrics.TemplatesUpdatedTotal.Add(ctx, 1,
		metric.WithAttributes(attribute.String("tenant_id", req.TenantId)))

	// Invalidate then re-warm cache
	if err := s.cache.Invalidate(ctx, tmpl.ID, tmpl.TenantID); err != nil {
		s.logger.Warn("cache invalidate failed (best-effort)", "error", err)
	}
	if err := s.cache.Set(ctx, tmpl); err != nil {
		s.logger.Warn("cache set after update failed (best-effort)", "error", err)
	}

	s.logger.Info("workflow template updated",
		"template_id", tmpl.ID,
		"tenant_id", tmpl.TenantID,
		"version", tmpl.Version,
	)
	return &pb.UpdateWorkflowTemplateResponse{Template: templateToProto(tmpl)}, nil
}

// ── ListTransitions ───────────────────────────────────────────────────────────

func (s *WorkflowService) ListTransitions(
	ctx context.Context,
	req *pb.ListTransitionsRequest,
) (*pb.ListTransitionsResponse, error) {
	if req.TemplateId == "" {
		return nil, status.Error(codes.InvalidArgument, "template_id is required")
	}
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	transitions, err := s.store.ListTransitions(ctx, req.TemplateId, req.TenantId)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "template %s not found", req.TemplateId)
		}
		s.logger.Error("ListTransitions store error", "template_id", req.TemplateId, "error", err)
		return nil, status.Errorf(codes.Internal, "list transitions: %v", err)
	}

	return &pb.ListTransitionsResponse{Transitions: transitionsToProto(transitions)}, nil
}

// ── EvaluateTransition ────────────────────────────────────────────────────────

func (s *WorkflowService) EvaluateTransition(
	ctx context.Context,
	req *pb.EvaluateTransitionRequest,
) (*pb.EvaluateTransitionResponse, error) {
	if req.TemplateId == "" {
		return nil, status.Error(codes.InvalidArgument, "template_id is required")
	}
	if req.TenantId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}
	if req.FromState == "" || req.ToState == "" {
		return nil, status.Error(codes.InvalidArgument, "from_state and to_state are required")
	}

	s.metrics.TransitionEvalTotal.Add(ctx, 1,
		metric.WithAttributes(attribute.String("tenant_id", req.TenantId)))

	// Resolve template (cache-first)
	tmpl, err := s.resolveTemplate(ctx, req.TemplateId, req.TenantId)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "template %s not found", req.TemplateId)
		}
		return nil, status.Errorf(codes.Internal, "resolve template: %v", err)
	}

	allowed, reason := evaluateTransition(tmpl, req.FromState, req.ToState, req.ActorRole)

	s.logger.Info("transition evaluated",
		"template_id", req.TemplateId,
		"from", req.FromState,
		"to", req.ToState,
		"actor_role", req.ActorRole,
		"allowed", allowed,
	)

	return &pb.EvaluateTransitionResponse{Allowed: allowed, Reason: reason}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (s *WorkflowService) resolveTemplate(ctx context.Context, templateID, tenantID string) (*db.WorkflowTemplate, error) {
	cached, err := s.cache.Get(ctx, templateID, tenantID)
	if err != nil {
		s.logger.Warn("cache get failed in resolveTemplate", "error", err)
	}
	if cached != nil {
		s.metrics.CacheHitsTotal.Add(ctx, 1)
		return cached, nil
	}
	s.metrics.CacheMissesTotal.Add(ctx, 1)

	tmpl, err := s.store.GetTemplate(ctx, templateID, tenantID)
	if err != nil {
		return nil, err
	}
	if setErr := s.cache.Set(ctx, tmpl); setErr != nil {
		s.logger.Warn("cache set failed (best-effort)", "error", setErr)
	}
	return tmpl, nil
}

// evaluateTransition checks whether a transition from→to is allowed for the given role.
func evaluateTransition(tmpl *db.WorkflowTemplate, fromState, toState, actorRole string) (bool, string) {
	for _, tr := range tmpl.Transitions {
		if tr.FromState != fromState || tr.ToState != toState {
			continue
		}
		// Check required_role at transition level
		if tr.RequiredRole != "" && tr.RequiredRole != actorRole {
			return false, fmt.Sprintf("role %q required for transition %s→%s, actor has %q",
				tr.RequiredRole, fromState, toState, actorRole)
		}
		// Check ROLE_REQUIRED guards
		for _, g := range tr.Guards {
			if g.Type == "GUARD_TYPE_ROLE_REQUIRED" && g.Role != "" && g.Role != actorRole {
				return false, fmt.Sprintf("guard requires role %q, actor has %q", g.Role, actorRole)
			}
		}
		return true, "transition allowed"
	}
	return false, fmt.Sprintf("no transition defined from %q to %q", fromState, toState)
}

// ── proto conversions ─────────────────────────────────────────────────────────

func templateToProto(t *db.WorkflowTemplate) *pb.WorkflowTemplate {
	return &pb.WorkflowTemplate{
		Id:          t.ID,
		TenantId:    t.TenantID,
		Name:        t.Name,
		Version:     int32(t.Version),
		States:      t.States,
		Transitions: transitionsToProto(t.Transitions),
		CreatedAt:   timestamppb.New(t.CreatedAt),
		UpdatedAt:   timestamppb.New(t.UpdatedAt),
	}
}

func transitionsToProto(transitions []db.Transition) []*pb.Transition {
	out := make([]*pb.Transition, len(transitions))
	for i, tr := range transitions {
		out[i] = &pb.Transition{
			FromState:    tr.FromState,
			ToState:      tr.ToState,
			RequiredRole: tr.RequiredRole,
			Guards:       guardsToProto(tr.Guards),
		}
	}
	return out
}

func guardsToProto(guards []db.Guard) []*pb.Guard {
	out := make([]*pb.Guard, len(guards))
	for i, g := range guards {
		out[i] = &pb.Guard{
			Type:      guardTypeToProto(g.Type),
			Role:      g.Role,
			Condition: g.Condition,
		}
	}
	return out
}

func guardTypeToProto(t string) pb.GuardType {
	switch t {
	case "GUARD_TYPE_ROLE_REQUIRED":
		return pb.GuardType_GUARD_TYPE_ROLE_REQUIRED
	case "GUARD_TYPE_CONDITION":
		return pb.GuardType_GUARD_TYPE_CONDITION
	default:
		return pb.GuardType_GUARD_TYPE_UNSPECIFIED
	}
}

func protoToTransitions(pbTransitions []*pb.Transition) []db.Transition {
	out := make([]db.Transition, len(pbTransitions))
	for i, tr := range pbTransitions {
		out[i] = db.Transition{
			FromState:    tr.FromState,
			ToState:      tr.ToState,
			RequiredRole: tr.RequiredRole,
			Guards:       protoToGuards(tr.Guards),
		}
	}
	return out
}

func protoToGuards(pbGuards []*pb.Guard) []db.Guard {
	out := make([]db.Guard, len(pbGuards))
	for i, g := range pbGuards {
		out[i] = db.Guard{
			Type:      g.Type.String(),
			Role:      g.Role,
			Condition: g.Condition,
		}
	}
	return out
}
