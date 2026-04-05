package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

// Guard represents a single transition guard condition.
type Guard struct {
	Type      string `json:"type"`
	Role      string `json:"role,omitempty"`
	Condition string `json:"condition,omitempty"`
}

// Transition represents a valid state change within a workflow template.
type Transition struct {
	ID           string
	TemplateID   string
	FromState    string
	ToState      string
	RequiredRole string
	Guards       []Guard
}

// WorkflowTemplate is the persisted representation of a workflow template.
type WorkflowTemplate struct {
	ID          string
	TenantID    string
	Name        string
	Version     int
	States      []string
	Transitions []Transition
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WorkflowStore provides all persistence operations for workflow templates.
type WorkflowStore struct {
	pool *pgxpool.Pool
}

func NewWorkflowStore(pool *pgxpool.Pool) *WorkflowStore {
	return &WorkflowStore{pool: pool}
}

// CreateTemplate inserts a new workflow template and its transitions atomically.
func (s *WorkflowStore) CreateTemplate(ctx context.Context, tenantID, name string, states []string, transitions []Transition) (*WorkflowTemplate, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var tmpl WorkflowTemplate
	err = tx.QueryRow(ctx,
		`INSERT INTO workflow_templates (tenant_id, name, states)
		 VALUES ($1, $2, $3)
		 RETURNING id, tenant_id, name, version, states, created_at, updated_at`,
		tenantID, name, states,
	).Scan(&tmpl.ID, &tmpl.TenantID, &tmpl.Name, &tmpl.Version, &tmpl.States, &tmpl.CreatedAt, &tmpl.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("template %q already exists for tenant: %w", name, ErrAlreadyExists)
		}
		return nil, fmt.Errorf("insert template: %w", err)
	}

	tmpl.Transitions, err = insertTransitions(ctx, tx, tmpl.ID, transitions)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &tmpl, nil
}

// GetTemplate retrieves a workflow template with its transitions by ID + tenantID.
func (s *WorkflowStore) GetTemplate(ctx context.Context, templateID, tenantID string) (*WorkflowTemplate, error) {
	var tmpl WorkflowTemplate
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, version, states, created_at, updated_at
		 FROM workflow_templates
		 WHERE id = $1 AND tenant_id = $2`,
		templateID, tenantID,
	).Scan(&tmpl.ID, &tmpl.TenantID, &tmpl.Name, &tmpl.Version, &tmpl.States, &tmpl.CreatedAt, &tmpl.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	tmpl.Transitions, err = s.listTransitions(ctx, tmpl.ID)
	if err != nil {
		return nil, err
	}
	return &tmpl, nil
}

// UpdateTemplate replaces a template's states and transitions, bumps version.
func (s *WorkflowStore) UpdateTemplate(ctx context.Context, templateID, tenantID string, states []string, transitions []Transition) (*WorkflowTemplate, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var tmpl WorkflowTemplate
	err = tx.QueryRow(ctx,
		`UPDATE workflow_templates
		 SET states = $1, version = version + 1, updated_at = NOW()
		 WHERE id = $2 AND tenant_id = $3
		 RETURNING id, tenant_id, name, version, states, created_at, updated_at`,
		states, templateID, tenantID,
	).Scan(&tmpl.ID, &tmpl.TenantID, &tmpl.Name, &tmpl.Version, &tmpl.States, &tmpl.CreatedAt, &tmpl.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update template: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM workflow_transitions WHERE template_id = $1`, tmpl.ID); err != nil {
		return nil, fmt.Errorf("delete old transitions: %w", err)
	}

	tmpl.Transitions, err = insertTransitions(ctx, tx, tmpl.ID, transitions)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &tmpl, nil
}

// ListTransitions returns transitions for a given template scoped to a tenant.
func (s *WorkflowStore) ListTransitions(ctx context.Context, templateID, tenantID string) ([]Transition, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM workflow_templates WHERE id = $1 AND tenant_id = $2)`,
		templateID, tenantID,
	).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check template ownership: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}
	return s.listTransitions(ctx, templateID)
}

// ── private helpers ────────────────────────────────────────────────────────────

func (s *WorkflowStore) listTransitions(ctx context.Context, templateID string) ([]Transition, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, template_id, from_state, to_state, required_role, guards
		 FROM workflow_transitions WHERE template_id = $1`,
		templateID,
	)
	if err != nil {
		return nil, fmt.Errorf("query transitions: %w", err)
	}
	defer rows.Close()

	var out []Transition
	for rows.Next() {
		var t Transition
		var guardsJSON []byte
		if err := rows.Scan(&t.ID, &t.TemplateID, &t.FromState, &t.ToState, &t.RequiredRole, &guardsJSON); err != nil {
			return nil, fmt.Errorf("scan transition: %w", err)
		}
		if err := json.Unmarshal(guardsJSON, &t.Guards); err != nil {
			return nil, fmt.Errorf("unmarshal guards: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func insertTransitions(ctx context.Context, tx pgx.Tx, templateID string, transitions []Transition) ([]Transition, error) {
	out := make([]Transition, 0, len(transitions))
	for _, tr := range transitions {
		guardsJSON, err := json.Marshal(tr.Guards)
		if err != nil {
			return nil, fmt.Errorf("marshal guards: %w", err)
		}
		var inserted Transition
		var rawGuards []byte
		err = tx.QueryRow(ctx,
			`INSERT INTO workflow_transitions (template_id, from_state, to_state, required_role, guards)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id, template_id, from_state, to_state, required_role, guards`,
			templateID, tr.FromState, tr.ToState, tr.RequiredRole, guardsJSON,
		).Scan(&inserted.ID, &inserted.TemplateID, &inserted.FromState, &inserted.ToState, &inserted.RequiredRole, &rawGuards)
		if err != nil {
			return nil, fmt.Errorf("insert transition: %w", err)
		}
		if err := json.Unmarshal(rawGuards, &inserted.Guards); err != nil {
			return nil, fmt.Errorf("unmarshal inserted guards: %w", err)
		}
		out = append(out, inserted)
	}
	return out, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

