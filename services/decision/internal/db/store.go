package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

// Decision is the persisted record of a risk decision.
type Decision struct {
	ID              string
	PaymentEventID  string
	TenantID        string
	Outcome         string
	ReasonCodes     []string
	ConfidenceScore float64
	RuleResults     json.RawMessage
	LatencyMs       int64
	Overridden      bool
	CreatedAt       time.Time
}

// DecisionOverride records an analyst manual override of a decision.
type DecisionOverride struct {
	ID              string
	DecisionID      string
	AnalystID       string
	PreviousOutcome string
	NewOutcome      string
	Reason          string
	CreatedAt       time.Time
}

type DecisionStore struct {
	pool *pgxpool.Pool
}

func NewDecisionStore(pool *pgxpool.Pool) *DecisionStore {
	return &DecisionStore{pool: pool}
}

// RecordDecision persists a new decision, ignoring duplicates (idempotent on payment_event_id+tenant_id).
func (s *DecisionStore) RecordDecision(
	ctx context.Context,
	paymentEventID, tenantID, outcome string,
	reasonCodes []string,
	confidenceScore float64,
	ruleResults json.RawMessage,
	latencyMs int64,
) (*Decision, error) {
	if reasonCodes == nil {
		reasonCodes = []string{}
	}
	if ruleResults == nil {
		ruleResults = json.RawMessage("[]")
	}

	var d Decision
	err := s.pool.QueryRow(ctx,
		`INSERT INTO decisions
		   (payment_event_id, tenant_id, outcome, reason_codes, confidence_score, rule_results, latency_ms)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (payment_event_id, tenant_id) DO NOTHING
		 RETURNING id, payment_event_id, tenant_id, outcome, reason_codes, confidence_score, rule_results, latency_ms, overridden, created_at`,
		paymentEventID, tenantID, outcome, reasonCodes, confidenceScore, []byte(ruleResults), latencyMs,
	).Scan(
		&d.ID, &d.PaymentEventID, &d.TenantID, &d.Outcome,
		&d.ReasonCodes, &d.ConfidenceScore, &d.RuleResults,
		&d.LatencyMs, &d.Overridden, &d.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// ON CONFLICT DO NOTHING returned no row — fetch the existing one
			return s.GetDecision(ctx, paymentEventID, tenantID)
		}
		return nil, fmt.Errorf("insert decision: %w", err)
	}
	return &d, nil
}

// GetDecision retrieves a decision by payment_event_id + tenant_id.
func (s *DecisionStore) GetDecision(ctx context.Context, paymentEventID, tenantID string) (*Decision, error) {
	var d Decision
	err := s.pool.QueryRow(ctx,
		`SELECT id, payment_event_id, tenant_id, outcome, reason_codes, confidence_score,
		        rule_results, latency_ms, overridden, created_at
		 FROM decisions
		 WHERE payment_event_id = $1 AND tenant_id = $2`,
		paymentEventID, tenantID,
	).Scan(
		&d.ID, &d.PaymentEventID, &d.TenantID, &d.Outcome,
		&d.ReasonCodes, &d.ConfidenceScore, &d.RuleResults,
		&d.LatencyMs, &d.Overridden, &d.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get decision: %w", err)
	}
	return &d, nil
}

// GetDecisionByID retrieves a decision by its UUID (needed for override).
func (s *DecisionStore) GetDecisionByID(ctx context.Context, id string) (*Decision, error) {
	var d Decision
	err := s.pool.QueryRow(ctx,
		`SELECT id, payment_event_id, tenant_id, outcome, reason_codes, confidence_score,
		        rule_results, latency_ms, overridden, created_at
		 FROM decisions WHERE id = $1`,
		id,
	).Scan(
		&d.ID, &d.PaymentEventID, &d.TenantID, &d.Outcome,
		&d.ReasonCodes, &d.ConfidenceScore, &d.RuleResults,
		&d.LatencyMs, &d.Overridden, &d.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get decision by id: %w", err)
	}
	return &d, nil
}

// OverrideDecision records an analyst override in a transaction.
func (s *DecisionStore) OverrideDecision(
	ctx context.Context,
	decisionID, analystID, newOutcome, reason string,
) (*DecisionOverride, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var prevOutcome string
	err = tx.QueryRow(ctx, `SELECT outcome FROM decisions WHERE id = $1 FOR UPDATE`, decisionID).
		Scan(&prevOutcome)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock decision: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE decisions SET outcome = $1, overridden = TRUE WHERE id = $2`,
		newOutcome, decisionID,
	); err != nil {
		return nil, fmt.Errorf("update decision outcome: %w", err)
	}

	var o DecisionOverride
	err = tx.QueryRow(ctx,
		`INSERT INTO decision_overrides (decision_id, analyst_id, previous_outcome, new_outcome, reason)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, decision_id, analyst_id, previous_outcome, new_outcome, reason, created_at`,
		decisionID, analystID, prevOutcome, newOutcome, reason,
	).Scan(&o.ID, &o.DecisionID, &o.AnalystID, &o.PreviousOutcome, &o.NewOutcome, &o.Reason, &o.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert override: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &o, nil
}

// ListDecisions returns paginated decisions for a tenant, optionally filtered by outcome.
func (s *DecisionStore) ListDecisions(
	ctx context.Context,
	tenantID string,
	page, pageSize int,
	outcomeFilter string,
) ([]Decision, int, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page < 0 {
		page = 0
	}
	offset := page * pageSize

	var total int
	var countErr error
	if outcomeFilter != "" {
		countErr = s.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM decisions WHERE tenant_id = $1 AND outcome = $2`,
			tenantID, outcomeFilter,
		).Scan(&total)
	} else {
		countErr = s.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM decisions WHERE tenant_id = $1`, tenantID).Scan(&total)
	}
	if countErr != nil {
		return nil, 0, fmt.Errorf("count decisions: %w", countErr)
	}

	var rows pgx.Rows
	var queryErr error
	if outcomeFilter != "" {
		rows, queryErr = s.pool.Query(ctx,
			`SELECT id, payment_event_id, tenant_id, outcome, reason_codes, confidence_score,
			        rule_results, latency_ms, overridden, created_at
			 FROM decisions
			 WHERE tenant_id = $1 AND outcome = $2
			 ORDER BY created_at DESC
			 LIMIT $3 OFFSET $4`,
			tenantID, outcomeFilter, pageSize, offset)
	} else {
		rows, queryErr = s.pool.Query(ctx,
			`SELECT id, payment_event_id, tenant_id, outcome, reason_codes, confidence_score,
			        rule_results, latency_ms, overridden, created_at
			 FROM decisions
			 WHERE tenant_id = $1
			 ORDER BY created_at DESC
			 LIMIT $2 OFFSET $3`,
			tenantID, pageSize, offset)
	}
	if queryErr != nil {
		return nil, 0, fmt.Errorf("list decisions: %w", queryErr)
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		var d Decision
		if err := rows.Scan(
			&d.ID, &d.PaymentEventID, &d.TenantID, &d.Outcome,
			&d.ReasonCodes, &d.ConfidenceScore, &d.RuleResults,
			&d.LatencyMs, &d.Overridden, &d.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan decision: %w", err)
		}
		decisions = append(decisions, d)
	}
	return decisions, total, rows.Err()
}
