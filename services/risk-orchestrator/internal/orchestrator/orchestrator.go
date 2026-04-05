package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"risk-orchestrator/internal/cache"
	"risk-orchestrator/internal/client"
	"risk-orchestrator/internal/telemetry"

	pb "risk-orchestrator/api/gen/rules-engine"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// RuleResultSummary is a per-rule outcome in the evaluation result.
type RuleResultSummary struct {
	RuleID   string
	RuleName string
	Matched  bool
	Action   string
	Reason   string
}

// RiskEvaluatedPayload is the data passed to the Publisher to emit to Kafka.
type RiskEvaluatedPayload struct {
	PaymentEventID  string
	TenantID        string
	AggregateAction string
	FailOpen        bool
	RuleResults     []RuleResultSummary
	LatencyMs       int64
	EvaluatedAt     time.Time
}

// Publisher is the interface for emitting risk evaluation results.
type Publisher interface {
	PublishRiskEvaluated(ctx context.Context, payload RiskEvaluatedPayload) error
}

// PaymentEvent is the normalised input to the orchestration pipeline.
type PaymentEvent struct {
	PaymentEventID string
	TenantID       string
	Amount         int64
	Currency       string
	Source         string
	Destination    string
	Metadata       map[string]string
}

// EvaluationResult is the output of the orchestration pipeline.
type EvaluationResult struct {
	PaymentEventID  string
	TenantID        string
	AggregateAction string // approve | flag | review | block
	FailOpen        bool
	RuleResults     []RuleResultSummary
	LatencyMs       int64
}

// Orchestrator coordinates risk evaluation for a payment event.
type Orchestrator struct {
	rulesClient *client.RulesClient
	cache       *cache.InFlightCache
	publisher   Publisher
	metrics     *telemetry.Metrics
	budget      time.Duration
	logger      *slog.Logger
}

func New(
	rulesClient *client.RulesClient,
	inFlight *cache.InFlightCache,
	publisher Publisher,
	metrics *telemetry.Metrics,
	budgetMs int64,
	logger *slog.Logger,
) *Orchestrator {
	return &Orchestrator{
		rulesClient: rulesClient,
		cache:       inFlight,
		publisher:   publisher,
		metrics:     metrics,
		budget:      time.Duration(budgetMs) * time.Millisecond,
		logger:      logger,
	}
}

// Process is the Kafka-driven entry point. It deduplicates via Redis and publishes results.
func (o *Orchestrator) Process(ctx context.Context, event PaymentEvent) error {
	log := o.logger.With("payment_event_id", event.PaymentEventID, "tenant_id", event.TenantID)

	acquired, err := o.cache.TryAcquire(ctx, event.PaymentEventID)
	if err != nil {
		log.Warn("in-flight cache error, proceeding without dedup", "error", err)
	} else if !acquired {
		log.Info("event already in-flight, skipping duplicate")
		return nil
	}
	defer func() {
		if releaseErr := o.cache.Release(ctx, event.PaymentEventID); releaseErr != nil {
			log.Warn("failed to release in-flight lock", "error", releaseErr)
		}
	}()

	result, err := o.run(ctx, event)
	if err != nil {
		return fmt.Errorf("orchestration failed: %w", err)
	}

	if pubErr := o.publish(ctx, result); pubErr != nil {
		log.Warn("publish risk.evaluated failed (best-effort)", "error", pubErr)
	}
	return nil
}

// Evaluate is the gRPC entry point — returns the result synchronously and also publishes to Kafka.
func (o *Orchestrator) Evaluate(ctx context.Context, event PaymentEvent) (*EvaluationResult, error) {
	result, err := o.run(ctx, event)
	if err != nil {
		return nil, err
	}
	if pubErr := o.publish(ctx, result); pubErr != nil {
		o.logger.Warn("publish risk.evaluated failed (best-effort)",
			"payment_event_id", event.PaymentEventID, "error", pubErr)
	}
	return result, nil
}

// run executes the core evaluation logic — calls Rules Engine with latency budget enforcement.
func (o *Orchestrator) run(ctx context.Context, event PaymentEvent) (*EvaluationResult, error) {
	log := o.logger.With("payment_event_id", event.PaymentEventID, "tenant_id", event.TenantID)

	ctx, span := telemetry.Tracer("orchestrator").Start(ctx, "Orchestrate",
		trace.WithAttributes(
			attribute.String("payment_event_id", event.PaymentEventID),
			attribute.String("tenant_id", event.TenantID),
		),
	)
	defer span.End()

	start := time.Now()

	evalCtx, cancel := context.WithTimeout(ctx, o.budget)
	defer cancel()

	resp, err := o.rulesClient.EvaluateRules(evalCtx, client.EvalRequest{
		PaymentEventID: event.PaymentEventID,
		TenantID:       event.TenantID,
		Amount:         event.Amount,
		Currency:       event.Currency,
		Source:         event.Source,
		Destination:    event.Destination,
		Metadata:       event.Metadata,
	})

	latencyMs := time.Since(start).Milliseconds()

	result := &EvaluationResult{
		PaymentEventID: event.PaymentEventID,
		TenantID:       event.TenantID,
		LatencyMs:      latencyMs,
	}

	if err != nil {
		log.Warn("rules engine call failed, applying fail-open",
			"error", err, "latency_ms", latencyMs)
		o.metrics.RulesEngineErrorsTotal.Add(ctx, 1)
		o.metrics.FailOpenTotal.Add(ctx, 1)
		result.AggregateAction = "approve"
		result.FailOpen = true
	} else {
		result.AggregateAction = actionStr(resp.AggregateAction)
		result.RuleResults = toRuleResultSummaries(resp.Results)
	}

	o.metrics.RiskEvaluationsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("aggregate_action", result.AggregateAction),
			attribute.Bool("fail_open", result.FailOpen),
		),
	)
	o.metrics.OrchestratorLatency.Record(ctx, float64(latencyMs)/1000.0)

	log.Info("risk evaluation complete",
		"aggregate_action", result.AggregateAction,
		"fail_open", result.FailOpen,
		"latency_ms", latencyMs)

	return result, nil
}

func (o *Orchestrator) publish(ctx context.Context, result *EvaluationResult) error {
	err := o.publisher.PublishRiskEvaluated(ctx, RiskEvaluatedPayload{
		PaymentEventID:  result.PaymentEventID,
		TenantID:        result.TenantID,
		AggregateAction: result.AggregateAction,
		FailOpen:        result.FailOpen,
		RuleResults:     result.RuleResults,
		LatencyMs:       result.LatencyMs,
		EvaluatedAt:     time.Now().UTC(),
	})
	if err == nil {
		o.metrics.KafkaPublishTotal.Add(ctx, 1)
	}
	return err
}

// actionStr converts a proto RuleAction enum to its string representation.
func actionStr(a pb.RuleAction) string {
	switch a {
	case pb.RuleAction_RULE_ACTION_FLAG:
		return "flag"
	case pb.RuleAction_RULE_ACTION_REVIEW:
		return "review"
	case pb.RuleAction_RULE_ACTION_BLOCK:
		return "block"
	default:
		return "approve"
	}
}

func toRuleResultSummaries(results []*pb.RuleResult) []RuleResultSummary {
	out := make([]RuleResultSummary, 0, len(results))
	for _, r := range results {
		out = append(out, RuleResultSummary{
			RuleID:   r.RuleId,
			RuleName: r.RuleName,
			Matched:  r.Matched,
			Action:   actionStr(r.Action),
			Reason:   r.Reason,
		})
	}
	return out
}
