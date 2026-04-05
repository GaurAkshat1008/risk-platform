package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

// Case is the persisted representation of a case.
type Case struct {
	ID             string
	DecisionID     string
	TenantID       string
	AssigneeID     string
	Status         string
	Priority       string
	PaymentEventID string
	Outcome        string
	SLADeadline    time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CaseAction records an action taken on a case (assign, status change, note).
type CaseAction struct {
	ID        string
	CaseID    string
	ActorID   string
	Action    string
	Notes     string
	CreatedAt time.Time
}

// CaseStore provides all persistence operations for cases and case actions.
type CaseStore struct {
	pool *pgxpool.Pool
}

func NewCaseStore(pool *pgxpool.Pool) *CaseStore {
	return &CaseStore{pool: pool}
}

// CreateCase inserts a new case, idempotent on (decision_id, tenant_id).
func (s *CaseStore) CreateCase(
	ctx context.Context,
	decisionID, tenantID, paymentEventID, outcome, priority string,
	slaDeadline time.Time,
) (*Case, error) {
	var c Case
	err := s.pool.QueryRow(ctx,
		`INSERT INTO cases
		   (decision_id, tenant_id, payment_event_id, outcome, priority, sla_deadline)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (decision_id, tenant_id) DO NOTHING
		 RETURNING id, decision_id, tenant_id, assignee_id, status, priority,
		           payment_event_id, outcome, sla_deadline, created_at, updated_at`,
		decisionID, tenantID, paymentEventID, outcome, priority, slaDeadline,
	).Scan(
		&c.ID, &c.DecisionID, &c.TenantID, &c.AssigneeID, &c.Status, &c.Priority,
		&c.PaymentEventID, &c.Outcome, &c.SLADeadline, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.GetCaseByDecision(ctx, decisionID, tenantID)
		}
		return nil, fmt.Errorf("insert case: %w", err)
	}
	return &c, nil
}

// GetCase retrieves a case by its UUID.
func (s *CaseStore) GetCase(ctx context.Context, caseID string) (*Case, error) {
	var c Case
	err := s.pool.QueryRow(ctx,
		`SELECT id, decision_id, tenant_id, assignee_id, status, priority,
		        payment_event_id, outcome, sla_deadline, created_at, updated_at
		 FROM cases WHERE id = $1`,
		caseID,
	).Scan(
		&c.ID, &c.DecisionID, &c.TenantID, &c.AssigneeID, &c.Status, &c.Priority,
		&c.PaymentEventID, &c.Outcome, &c.SLADeadline, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get case: %w", err)
	}
	return &c, nil
}

// GetCaseByDecision retrieves a case by decision_id + tenant_id.
func (s *CaseStore) GetCaseByDecision(ctx context.Context, decisionID, tenantID string) (*Case, error) {
	var c Case
	err := s.pool.QueryRow(ctx,
		`SELECT id, decision_id, tenant_id, assignee_id, status, priority,
		        payment_event_id, outcome, sla_deadline, created_at, updated_at
		 FROM cases WHERE decision_id = $1 AND tenant_id = $2`,
		decisionID, tenantID,
	).Scan(
		&c.ID, &c.DecisionID, &c.TenantID, &c.AssigneeID, &c.Status, &c.Priority,
		&c.PaymentEventID, &c.Outcome, &c.SLADeadline, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get case by decision: %w", err)
	}
	return &c, nil
}

// AssignCase sets the assignee and records a case_action, all within a transaction.
func (s *CaseStore) AssignCase(ctx context.Context, caseID, assigneeID, actorID, notes string) (*Case, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var c Case
	err = tx.QueryRow(ctx,
		`UPDATE cases
		 SET assignee_id = $2, status = 'in_review', updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, decision_id, tenant_id, assignee_id, status, priority,
		           payment_event_id, outcome, sla_deadline, created_at, updated_at`,
		caseID, assigneeID,
	).Scan(
		&c.ID, &c.DecisionID, &c.TenantID, &c.AssigneeID, &c.Status, &c.Priority,
		&c.PaymentEventID, &c.Outcome, &c.SLADeadline, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("assign case: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO case_actions (case_id, actor_id, action, notes) VALUES ($1, $2, 'assign', $3)`,
		caseID, actorID, notes,
	); err != nil {
		return nil, fmt.Errorf("insert case_action: %w", err)
	}

	return &c, tx.Commit(ctx)
}

// UpdateCaseStatus changes the status of a case and records a case_action.
func (s *CaseStore) UpdateCaseStatus(ctx context.Context, caseID, newStatus, actorID, notes string) (*Case, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var c Case
	err = tx.QueryRow(ctx,
		`UPDATE cases
		 SET status = $2, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, decision_id, tenant_id, assignee_id, status, priority,
		           payment_event_id, outcome, sla_deadline, created_at, updated_at`,
		caseID, newStatus,
	).Scan(
		&c.ID, &c.DecisionID, &c.TenantID, &c.AssigneeID, &c.Status, &c.Priority,
		&c.PaymentEventID, &c.Outcome, &c.SLADeadline, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update case status: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO case_actions (case_id, actor_id, action, notes) VALUES ($1, $2, $3, $4)`,
		caseID, actorID, newStatus, notes,
	); err != nil {
		return nil, fmt.Errorf("insert case_action: %w", err)
	}

	return &c, tx.Commit(ctx)
}

// EscalateCase transitions a case to escalated status.
func (s *CaseStore) EscalateCase(ctx context.Context, caseID, actorID, reason string) (*Case, error) {
	return s.UpdateCaseStatus(ctx, caseID, "escalated", actorID, reason)
}

// ListCases returns paginated cases for a tenant with optional filters.
func (s *CaseStore) ListCases(
	ctx context.Context,
	tenantID, statusFilter, assigneeFilter string,
	pageSize int,
	offset int,
) ([]Case, error) {
	if pageSize <= 0 {
		pageSize = 20
	}

	query := `SELECT id, decision_id, tenant_id, assignee_id, status, priority,
	                 payment_event_id, outcome, sla_deadline, created_at, updated_at
	          FROM cases
	          WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2

	if statusFilter != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, statusFilter)
		argIdx++
	}
	if assigneeFilter != "" {
		query += fmt.Sprintf(" AND assignee_id = $%d", argIdx)
		args = append(args, assigneeFilter)
		argIdx++
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list cases: %w", err)
	}
	defer rows.Close()

	var cases []Case
	for rows.Next() {
		var c Case
		if err := rows.Scan(
			&c.ID, &c.DecisionID, &c.TenantID, &c.AssigneeID, &c.Status, &c.Priority,
			&c.PaymentEventID, &c.Outcome, &c.SLADeadline, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan case: %w", err)
		}
		cases = append(cases, c)
	}
	return cases, rows.Err()
}

// GetOpenCasesBreachedSLA returns open/in_review cases whose SLA deadline has passed.
func (s *CaseStore) GetOpenCasesBreachedSLA(ctx context.Context) ([]Case, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, decision_id, tenant_id, assignee_id, status, priority,
		        payment_event_id, outcome, sla_deadline, created_at, updated_at
		 FROM cases
		 WHERE status IN ('open', 'in_review') AND sla_deadline < NOW()`,
	)
	if err != nil {
		return nil, fmt.Errorf("get sla breached cases: %w", err)
	}
	defer rows.Close()

	var cases []Case
	for rows.Next() {
		var c Case
		if err := rows.Scan(
			&c.ID, &c.DecisionID, &c.TenantID, &c.AssigneeID, &c.Status, &c.Priority,
			&c.PaymentEventID, &c.Outcome, &c.SLADeadline, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan case: %w", err)
		}
		cases = append(cases, c)
	}
	return cases, rows.Err()
}
