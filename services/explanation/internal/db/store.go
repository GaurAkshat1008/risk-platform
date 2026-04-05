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

// RuleContribution records a single rule's influence on a decision.
type RuleContribution struct {
	RuleID   string `json:"rule_id"`
	RuleName string `json:"rule_name"`
	Matched  bool   `json:"matched"`
	Action   string `json:"action"`
	Reason   string `json:"reason"`
}

// FeatureValue captures a named feature and the value it held at decision time.
type FeatureValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Explanation is the persisted record of a decision's rationale.
type Explanation struct {
	ID             string
	DecisionID     string
	TenantID       string
	PaymentEventID string
	Outcome        string
	ConfidenceScore float64
	RuleContributions []RuleContribution
	FeatureValues     []FeatureValue
	Narrative      string
	PolicyVersion  string
	GeneratedAt    time.Time
}

// ExplanationStore provides persistence for decision explanations.
type ExplanationStore struct {
	pool *pgxpool.Pool
}

func NewExplanationStore(pool *pgxpool.Pool) *ExplanationStore {
	return &ExplanationStore{pool: pool}
}

// Upsert inserts or replaces an explanation for a given decision_id + tenant_id.
// The decision_id column has a UNIQUE constraint so concurrent callers converge.
func (s *ExplanationStore) Upsert(ctx context.Context, e *Explanation) error {
	rules, err := json.Marshal(e.RuleContributions)
	if err != nil {
		return fmt.Errorf("marshal rule contributions: %w", err)
	}
	features, err := json.Marshal(e.FeatureValues)
	if err != nil {
		return fmt.Errorf("marshal feature values: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO explanations
			(decision_id, tenant_id, payment_event_id, outcome, confidence_score,
			 rule_contributions, feature_values, narrative, policy_version, generated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (decision_id) DO UPDATE
			SET rule_contributions = EXCLUDED.rule_contributions,
			    feature_values     = EXCLUDED.feature_values,
			    narrative          = EXCLUDED.narrative,
			    policy_version     = EXCLUDED.policy_version,
			    generated_at       = EXCLUDED.generated_at`,
		e.DecisionID, e.TenantID, e.PaymentEventID, e.Outcome, e.ConfidenceScore,
		rules, features, e.Narrative, e.PolicyVersion,
	)
	return err
}

// GetByPaymentEvent fetches the explanation for a payment_event_id + tenant_id pair.
func (s *ExplanationStore) GetByPaymentEvent(ctx context.Context, paymentEventID, tenantID string) (*Explanation, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, decision_id, tenant_id, payment_event_id, outcome, confidence_score,
		       rule_contributions, feature_values, narrative, policy_version, generated_at
		FROM explanations
		WHERE payment_event_id = $1 AND tenant_id = $2`,
		paymentEventID, tenantID,
	)

	var e Explanation
	var rulesRaw, featuresRaw []byte
	err := row.Scan(
		&e.ID, &e.DecisionID, &e.TenantID, &e.PaymentEventID, &e.Outcome, &e.ConfidenceScore,
		&rulesRaw, &featuresRaw, &e.Narrative, &e.PolicyVersion, &e.GeneratedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query explanation: %w", err)
	}

	if err := json.Unmarshal(rulesRaw, &e.RuleContributions); err != nil {
		return nil, fmt.Errorf("unmarshal rule contributions: %w", err)
	}
	if err := json.Unmarshal(featuresRaw, &e.FeatureValues); err != nil {
		return nil, fmt.Errorf("unmarshal feature values: %w", err)
	}
	return &e, nil
}
