package grpc

import (
    "context"
    "errors"
    "log/slog"
    "time"

    pb "rules-engine/api/gen/rules-engine"
    "rules-engine/internal/cache"
    "rules-engine/internal/db"
    "rules-engine/internal/engine"
    "rules-engine/internal/kafka"
    "rules-engine/internal/telemetry"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/structpb"
    "google.golang.org/protobuf/types/known/timestamppb"
)

// actionPriority maps action strings to severity scores for aggregate action selection.
var actionPriority = map[string]int{
    "approve": 1,
    "flag":    2,
    "review":  3,
    "block":   4,
}

// actionFromProto converts pb.RuleAction to its string representation.
var actionFromProto = map[pb.RuleAction]string{
    pb.RuleAction_RULE_ACTION_APPROVE: "approve",
    pb.RuleAction_RULE_ACTION_FLAG:    "flag",
    pb.RuleAction_RULE_ACTION_REVIEW:  "review",
    pb.RuleAction_RULE_ACTION_BLOCK:   "block",
}

// actionToProto converts a string action to pb.RuleAction.
var actionToProto = map[string]pb.RuleAction{
    "approve": pb.RuleAction_RULE_ACTION_APPROVE,
    "flag":    pb.RuleAction_RULE_ACTION_FLAG,
    "review":  pb.RuleAction_RULE_ACTION_REVIEW,
    "block":   pb.RuleAction_RULE_ACTION_BLOCK,
}

type RulesEngineService struct {
    pb.UnimplementedRulesEngineServiceServer
    store     *db.RuleStore
    ruleCache *cache.RuleCache
    publisher *kafka.RuleEventPublisher
    metrics   *telemetry.Metrics
    logger    *slog.Logger
}

func NewRulesEngineService(
    store *db.RuleStore,
    ruleCache *cache.RuleCache,
    publisher *kafka.RuleEventPublisher,
    metrics *telemetry.Metrics,
    logger *slog.Logger,
) *RulesEngineService {
    return &RulesEngineService{
        store:     store,
        ruleCache: ruleCache,
        publisher: publisher,
        metrics:   metrics,
        logger:    logger,
    }
}

// ── EvaluateRules ─────────────────────────────────────────────────────────────

func (s *RulesEngineService) EvaluateRules(ctx context.Context, req *pb.EvaluateRulesRequest) (*pb.EvaluateRulesResponse, error) {
    if req.Context == nil || req.Context.TenantId == "" {
        return nil, status.Error(codes.InvalidArgument, "context.tenant_id is required")
    }
    c := req.Context
    log := s.logger.With("tenant_id", c.TenantId, "payment_event_id", c.PaymentEventId)

    evalCtx := engine.PaymentContext{
        PaymentEventID: c.PaymentEventId,
        TenantID:       c.TenantId,
        Amount:         c.Amount,
        Currency:       c.Currency,
        Source:         c.Source,
        Destination:    c.Destination,
        Metadata:       c.Metadata,
    }

    rules, err := s.loadRules(ctx, c.TenantId)
    if err != nil {
        log.Error("failed to load rules", "error", err)
        return nil, status.Errorf(codes.Internal, "load rules: %v", err)
    }

    var pbResults []*pb.RuleResult
    aggregateAction := "approve"

    start := time.Now()
    for _, r := range rules {
        result, err := engine.Evaluate(r.Expression, evalCtx)
        if err != nil {
            log.Warn("rule evaluation error", "rule_id", r.ID, "error", err)
            continue
        }

        pbResult := &pb.RuleResult{
            RuleId:      r.ID,
            RuleName:    r.Name,
            Matched:     result.Matched,
            Action:      actionToProto[r.Action],
            Reason:      result.Reason,
            EvaluatedAt: timestamppb.Now(),
        }
        pbResults = append(pbResults, pbResult)

        if result.Matched {
            s.metrics.RulesMatchedTotal.Add(ctx, 1,
                metric.WithAttributes(attribute.String("tenant_id", c.TenantId)))
            if actionPriority[r.Action] > actionPriority[aggregateAction] {
                aggregateAction = r.Action
            }
        }
    }
    s.metrics.EvaluationDuration.Record(ctx, time.Since(start).Seconds(),
        metric.WithAttributes(attribute.String("tenant_id", c.TenantId)))

    // Publish best-effort
    kafkaResults := toKafkaResults(pbResults)
    if err := s.publisher.PublishRulesEvaluated(ctx, c.TenantId, c.PaymentEventId, aggregateAction, kafkaResults); err != nil {
        log.Warn("publish rules.evaluated failed", "error", err)
    } else {
        s.metrics.KafkaPublishTotal.Add(ctx, 1)
    }

    log.Info("rules evaluated",
        "rules_count", len(rules),
        "matched_count", countMatched(pbResults),
        "aggregate_action", aggregateAction,
    )

    return &pb.EvaluateRulesResponse{
        Results:         pbResults,
        AggregateAction: actionToProto[aggregateAction],
    }, nil
}

// ── CreateRule ────────────────────────────────────────────────────────────────

func (s *RulesEngineService) CreateRule(ctx context.Context, req *pb.CreateRuleRequest) (*pb.CreateRuleResponse, error) {
    if req.TenantId == "" || req.Name == "" || req.Expression == nil {
        return nil, status.Error(codes.InvalidArgument, "tenant_id, name, and expression are required")
    }
    action := actionFromProto[req.Action]
    if action == "" {
        action = "flag"
    }

    r, err := s.store.CreateRule(ctx, req.TenantId, req.Name, req.Expression.AsMap(), action, req.Priority)
    if err != nil {
        if errors.Is(err, db.ErrAlreadyExists) {
            return nil, status.Errorf(codes.AlreadyExists, "rule %q already exists for tenant", req.Name)
        }
        s.logger.Error("CreateRule failed", "tenant_id", req.TenantId, "error", err)
        return nil, status.Errorf(codes.Internal, "create rule: %v", err)
    }

    if err := s.ruleCache.Invalidate(ctx, req.TenantId); err != nil {
        s.logger.Warn("cache invalidate failed", "tenant_id", req.TenantId, "error", err)
    }

    s.logger.Info("rule created", "rule_id", r.ID, "tenant_id", r.TenantID, "name", r.Name)
    return &pb.CreateRuleResponse{Rule: toProtoRule(r)}, nil
}

// ── UpdateRule ────────────────────────────────────────────────────────────────

func (s *RulesEngineService) UpdateRule(ctx context.Context, req *pb.UpdateRuleRequest) (*pb.UpdateRuleResponse, error) {
    if req.RuleId == "" || req.TenantId == "" || req.Expression == nil {
        return nil, status.Error(codes.InvalidArgument, "rule_id, tenant_id, and expression are required")
    }
    action := actionFromProto[req.Action]
    if action == "" {
        action = "flag"
    }

    r, err := s.store.UpdateRule(ctx, req.RuleId, req.TenantId, req.Expression.AsMap(), action, req.Priority, req.Enabled)
    if err != nil {
        if errors.Is(err, db.ErrNotFound) {
            return nil, status.Error(codes.NotFound, "rule not found")
        }
        s.logger.Error("UpdateRule failed", "rule_id", req.RuleId, "error", err)
        return nil, status.Errorf(codes.Internal, "update rule: %v", err)
    }

    if err := s.ruleCache.Invalidate(ctx, req.TenantId); err != nil {
        s.logger.Warn("cache invalidate failed", "tenant_id", req.TenantId, "error", err)
    }

    s.logger.Info("rule updated", "rule_id", r.ID, "version", r.Version)
    return &pb.UpdateRuleResponse{Rule: toProtoRule(r)}, nil
}

// ── DeleteRule ────────────────────────────────────────────────────────────────

func (s *RulesEngineService) DeleteRule(ctx context.Context, req *pb.DeleteRuleRequest) (*pb.DeleteRuleResponse, error) {
    if req.RuleId == "" || req.TenantId == "" {
        return nil, status.Error(codes.InvalidArgument, "rule_id and tenant_id are required")
    }

    if err := s.store.DeleteRule(ctx, req.RuleId, req.TenantId); err != nil {
        if errors.Is(err, db.ErrNotFound) {
            return nil, status.Error(codes.NotFound, "rule not found")
        }
        s.logger.Error("DeleteRule failed", "rule_id", req.RuleId, "error", err)
        return nil, status.Errorf(codes.Internal, "delete rule: %v", err)
    }

    if err := s.ruleCache.Invalidate(ctx, req.TenantId); err != nil {
        s.logger.Warn("cache invalidate failed", "tenant_id", req.TenantId, "error", err)
    }

    s.logger.Info("rule deleted", "rule_id", req.RuleId, "tenant_id", req.TenantId)
    return &pb.DeleteRuleResponse{Success: true}, nil
}

// ── ListRules ─────────────────────────────────────────────────────────────────

func (s *RulesEngineService) ListRules(ctx context.Context, req *pb.ListRulesRequest) (*pb.ListRulesResponse, error) {
    if req.TenantId == "" {
        return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
    }

    rules, err := s.store.ListRules(ctx, req.TenantId, req.IncludeDisabled)
    if err != nil {
        s.logger.Error("ListRules failed", "tenant_id", req.TenantId, "error", err)
        return nil, status.Errorf(codes.Internal, "list rules: %v", err)
    }

    var pbRules []*pb.Rule
    for _, r := range rules {
        r := r
        pbRules = append(pbRules, toProtoRule(&r))
    }
    return &pb.ListRulesResponse{Rules: pbRules}, nil
}

// ── SimulateRule ──────────────────────────────────────────────────────────────

func (s *RulesEngineService) SimulateRule(ctx context.Context, req *pb.SimulateRuleRequest) (*pb.SimulateRuleResponse, error) {
    if req.TenantId == "" || req.Context == nil {
        return nil, status.Error(codes.InvalidArgument, "tenant_id and context are required")
    }

    var expression map[string]any
    var action string
    var ruleName string

    if req.RuleId != "" {
        // Load an existing rule
        r, err := s.store.GetRule(ctx, req.RuleId, req.TenantId)
        if err != nil {
            if errors.Is(err, db.ErrNotFound) {
                return nil, status.Error(codes.NotFound, "rule not found")
            }
            return nil, status.Errorf(codes.Internal, "get rule: %v", err)
        }
        expression = r.Expression
        action = r.Action
        ruleName = r.Name
    } else {
        // Use the provided expression
        if req.Expression == nil {
            return nil, status.Error(codes.InvalidArgument, "either rule_id or expression must be provided")
        }
        expression = req.Expression.AsMap()
        action = actionFromProto[req.Action]
        if action == "" {
            action = "flag"
        }
        ruleName = "simulation"
    }

    c := req.Context
    evalCtx := engine.PaymentContext{
        PaymentEventID: c.PaymentEventId,
        TenantID:       c.TenantId,
        Amount:         c.Amount,
        Currency:       c.Currency,
        Source:         c.Source,
        Destination:    c.Destination,
        Metadata:       c.Metadata,
    }

    result, err := engine.Evaluate(expression, evalCtx)
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "evaluate expression: %v", err)
    }

    return &pb.SimulateRuleResponse{
        Result: &pb.RuleResult{
            RuleId:      req.RuleId,
            RuleName:    ruleName,
            Matched:     result.Matched,
            Action:      actionToProto[action],
            Reason:      result.Reason,
            EvaluatedAt: timestamppb.Now(),
        },
    }, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// loadRules fetches from cache first, falls back to DB and warms the cache.
func (s *RulesEngineService) loadRules(ctx context.Context, tenantID string) ([]cache.CachedRule, error) {
    cached, err := s.ruleCache.GetRules(ctx, tenantID)
    if err != nil {
        s.logger.Warn("cache get failed, falling through to DB", "tenant_id", tenantID, "error", err)
    }
    if cached != nil {
        return cached, nil
    }

    rules, err := s.store.ListRules(ctx, tenantID, false)
    if err != nil {
        return nil, err
    }

    var cr []cache.CachedRule
    for _, r := range rules {
        cr = append(cr, cache.CachedRule{
            ID:         r.ID,
            Name:       r.Name,
            Version:    r.Version,
            Expression: r.Expression,
            Action:     r.Action,
            Priority:   r.Priority,
        })
    }

    if err := s.ruleCache.SetRules(ctx, tenantID, cr); err != nil {
        s.logger.Warn("cache set failed", "tenant_id", tenantID, "error", err)
    }
    return cr, nil
}

func toProtoRule(r *db.Rule) *pb.Rule {
    exprStruct, _ := structpb.NewStruct(r.Expression)
    return &pb.Rule{
        Id:        r.ID,
        TenantId:  r.TenantID,
        Name:      r.Name,
        Version:   r.Version,
        Expression: exprStruct,
        Action:    actionToProto[r.Action],
        Priority:  r.Priority,
        Enabled:   r.Enabled,
        CreatedAt: timestamppb.New(r.CreatedAt),
        UpdatedAt: timestamppb.New(r.UpdatedAt),
    }
}

func toKafkaResults(results []*pb.RuleResult) []kafka.RuleEvalResult {
    out := make([]kafka.RuleEvalResult, 0, len(results))
    for _, r := range results {
        out = append(out, kafka.RuleEvalResult{
            RuleID:   r.RuleId,
            RuleName: r.RuleName,
            Matched:  r.Matched,
            Action:   r.Action.String(),
            Reason:   r.Reason,
        })
    }
    return out
}

func countMatched(results []*pb.RuleResult) int {
    n := 0
    for _, r := range results {
        if r.Matched {
            n++
        }
    }
    return n
}