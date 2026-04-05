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

type Rule struct {
    ID         string
    TenantID   string
    Name       string
    Version    int32
    Expression map[string]any
    Action     string
    Priority   int32
    Enabled    bool
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type RuleStore struct {
    pool *pgxpool.Pool
}

func NewRuleStore(pool *pgxpool.Pool) *RuleStore {
    return &RuleStore{pool: pool}
}

func (s *RuleStore) CreateRule(ctx context.Context, tenantID, name string, expression map[string]any, action string, priority int32) (*Rule, error) {
    exprJSON, err := json.Marshal(expression)
    if err != nil {
        return nil, fmt.Errorf("marshal expression: %w", err)
    }

    tx, err := s.pool.Begin(ctx)
    if err != nil {
        return nil, fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback(ctx)

    var r Rule
    var exprBytes []byte
    err = tx.QueryRow(ctx,
        `INSERT INTO rules (tenant_id, name, expression, action, priority)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, tenant_id, name, version, expression, action, priority, enabled, created_at, updated_at`,
        tenantID, name, exprJSON, action, priority,
    ).Scan(&r.ID, &r.TenantID, &r.Name, &r.Version, &exprBytes, &r.Action, &r.Priority, &r.Enabled, &r.CreatedAt, &r.UpdatedAt)
    if err != nil {
        if isUniqueViolation(err) {
            return nil, fmt.Errorf("rule with name %q already exists for tenant: %w", name, ErrAlreadyExists)
        }
        return nil, fmt.Errorf("insert rule: %w", err)
    }

    if _, err := tx.Exec(ctx,
        `INSERT INTO rule_versions (rule_id, version, expression, action) VALUES ($1, $2, $3, $4)`,
        r.ID, r.Version, exprJSON, action,
    ); err != nil {
        return nil, fmt.Errorf("insert rule_version: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return nil, fmt.Errorf("commit: %w", err)
    }

    _ = json.Unmarshal(exprBytes, &r.Expression)
    return &r, nil
}

func (s *RuleStore) GetRule(ctx context.Context, ruleID, tenantID string) (*Rule, error) {
    var r Rule
    var exprBytes []byte
    err := s.pool.QueryRow(ctx,
        `SELECT id, tenant_id, name, version, expression, action, priority, enabled, created_at, updated_at
         FROM rules WHERE id = $1 AND tenant_id = $2`,
        ruleID, tenantID,
    ).Scan(&r.ID, &r.TenantID, &r.Name, &r.Version, &exprBytes, &r.Action, &r.Priority, &r.Enabled, &r.CreatedAt, &r.UpdatedAt)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, fmt.Errorf("rule not found: %w", ErrNotFound)
    }
    if err != nil {
        return nil, fmt.Errorf("get rule: %w", err)
    }
    _ = json.Unmarshal(exprBytes, &r.Expression)
    return &r, nil
}

func (s *RuleStore) ListRules(ctx context.Context, tenantID string, includeDisabled bool) ([]Rule, error) {
    query := `SELECT id, tenant_id, name, version, expression, action, priority, enabled, created_at, updated_at
              FROM rules WHERE tenant_id = $1`
    if !includeDisabled {
        query += ` AND enabled = true`
    }
    query += ` ORDER BY priority DESC, created_at ASC`

    rows, err := s.pool.Query(ctx, query, tenantID)
    if err != nil {
        return nil, fmt.Errorf("list rules: %w", err)
    }
    defer rows.Close()

    var rules []Rule
    for rows.Next() {
        var r Rule
        var exprBytes []byte
        if err := rows.Scan(&r.ID, &r.TenantID, &r.Name, &r.Version, &exprBytes, &r.Action, &r.Priority, &r.Enabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
            return nil, fmt.Errorf("scan rule: %w", err)
        }
        _ = json.Unmarshal(exprBytes, &r.Expression)
        rules = append(rules, r)
    }
    return rules, rows.Err()
}

func (s *RuleStore) UpdateRule(ctx context.Context, ruleID, tenantID string, expression map[string]any, action string, priority int32, enabled bool) (*Rule, error) {
    exprJSON, err := json.Marshal(expression)
    if err != nil {
        return nil, fmt.Errorf("marshal expression: %w", err)
    }

    tx, err := s.pool.Begin(ctx)
    if err != nil {
        return nil, fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback(ctx)

    // Archive current version before overwriting.
    var currentVersion int32
    var currentExprBytes []byte
    var currentAction string
    err = tx.QueryRow(ctx,
        `SELECT version, expression, action FROM rules WHERE id = $1 AND tenant_id = $2`,
        ruleID, tenantID,
    ).Scan(&currentVersion, &currentExprBytes, &currentAction)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, fmt.Errorf("rule not found: %w", ErrNotFound)
    }
    if err != nil {
        return nil, fmt.Errorf("fetch current rule: %w", err)
    }

    if _, err := tx.Exec(ctx,
        `INSERT INTO rule_versions (rule_id, version, expression, action) VALUES ($1, $2, $3, $4)`,
        ruleID, currentVersion, currentExprBytes, currentAction,
    ); err != nil {
        return nil, fmt.Errorf("archive rule_version: %w", err)
    }

    var r Rule
    var exprBytes []byte
    err = tx.QueryRow(ctx,
        `UPDATE rules
         SET expression = $1, action = $2, priority = $3, enabled = $4,
             version = version + 1, updated_at = now()
         WHERE id = $5 AND tenant_id = $6
         RETURNING id, tenant_id, name, version, expression, action, priority, enabled, created_at, updated_at`,
        exprJSON, action, priority, enabled, ruleID, tenantID,
    ).Scan(&r.ID, &r.TenantID, &r.Name, &r.Version, &exprBytes, &r.Action, &r.Priority, &r.Enabled, &r.CreatedAt, &r.UpdatedAt)
    if err != nil {
        return nil, fmt.Errorf("update rule: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return nil, fmt.Errorf("commit: %w", err)
    }

    _ = json.Unmarshal(exprBytes, &r.Expression)
    return &r, nil
}

func (s *RuleStore) DeleteRule(ctx context.Context, ruleID, tenantID string) error {
    tag, err := s.pool.Exec(ctx,
        `DELETE FROM rules WHERE id = $1 AND tenant_id = $2`,
        ruleID, tenantID,
    )
    if err != nil {
        return fmt.Errorf("delete rule: %w", err)
    }
    if tag.RowsAffected() == 0 {
        return fmt.Errorf("rule not found: %w", ErrNotFound)
    }
    return nil
}

// ErrNotFound is a sentinel for not-found DB results.
var ErrNotFound = errors.New("not found")

// ErrAlreadyExists is a sentinel for unique constraint violations.
var ErrAlreadyExists = errors.New("already exists")

func isUniqueViolation(err error) bool {
    var pgErr *pgconn.PgError
    return errors.As(err, &pgErr) && pgErr.Code == "23505"
}