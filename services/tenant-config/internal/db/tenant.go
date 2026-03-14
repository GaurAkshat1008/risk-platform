package db

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

type TenantStatus string

const (
    TenantStatusActive     TenantStatus = "active"
    TenantStatusSuspended  TenantStatus = "suspended"
    TenantStatusOnboarding TenantStatus = "onboarding"
)

type Tenant struct {
    ID        string
    Name      string
    Status    TenantStatus
    CreatedAt time.Time
    UpdatedAt time.Time
}

type TenantConfig struct {
    ID                 string
    TenantID           string
    RuleSetId          string
    WorkflowTemplateID string
    Metadata           map[string]string
    Version            int32
    CreatedAt          time.Time
    UpdatedAt          time.Time
}

type FeatureFlag struct {
    ID                string
    TenantID          string
    Key               string
    Enabled           bool
    RolloutPercentage int32
    CreatedAt         time.Time
    UpdatedAt         time.Time
}

type TenantStore struct {
    pool *pgxpool.Pool
}

func NewTenantStore(pool *pgxpool.Pool) *TenantStore {
    return &TenantStore{pool: pool}
}

func (s *TenantStore) CreateTenant(ctx context.Context, name, ruleSetID, workflowTemplateID string, flags []FeatureFlag, metadata map[string]string) (*Tenant, *TenantConfig, error) {
    tx, err := s.pool.Begin(ctx)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback(ctx)

    var t Tenant
    if err := tx.QueryRow(ctx,
        `INSERT INTO tenants (name, status) VALUES ($1, $2) RETURNING id, name, status, created_at, updated_at`,
        name, TenantStatusOnboarding,
    ).Scan(&t.ID, &t.Name, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
        return nil, nil, fmt.Errorf("failed to insert tenant: %w", err)
    }

    metaJSON, _ := json.Marshal(metadata)
    var cfg TenantConfig
    var metaBytes []byte
    if err := tx.QueryRow(ctx,
        `INSERT INTO tenant_configs (tenant_id, rule_set_id, workflow_template_id, metadata, version)
         VALUES ($1, $2, $3, $4, 1)
         RETURNING id, tenant_id, rule_set_id, workflow_template_id, metadata, version, created_at, updated_at`,
        t.ID, ruleSetID, workflowTemplateID, metaJSON,
    ).Scan(&cfg.ID, &cfg.TenantID, &cfg.RuleSetId, &cfg.WorkflowTemplateID, &metaBytes, &cfg.Version, &cfg.CreatedAt, &cfg.UpdatedAt); err != nil {
        return nil, nil, fmt.Errorf("failed to insert tenant config: %w", err)
    }
    if len(metaBytes) > 0 {
        _ = json.Unmarshal(metaBytes, &cfg.Metadata)
    }

    for _, flag := range flags {
        if _, err := tx.Exec(ctx,
            `INSERT INTO feature_flags (tenant_id, key, enabled, rollout_percentage) VALUES ($1, $2, $3, $4)
             ON CONFLICT (tenant_id, key) DO UPDATE
               SET enabled = EXCLUDED.enabled, rollout_percentage = EXCLUDED.rollout_percentage, updated_at = NOW()`,
            t.ID, flag.Key, flag.Enabled, flag.RolloutPercentage,
        ); err != nil {
            return nil, nil, fmt.Errorf("failed to insert feature flag: %w", err)
        }
    }

    if err := tx.Commit(ctx); err != nil {
        return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
    }
    return &t, &cfg, nil
}

func (s *TenantStore) GetTenant(ctx context.Context, tenantID string) (*Tenant, *TenantConfig, []FeatureFlag, error) {
    var t Tenant
    if err := s.pool.QueryRow(ctx,
        `SELECT id, name, status, created_at, updated_at FROM tenants WHERE id = $1`,
        tenantID,
    ).Scan(&t.ID, &t.Name, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
        return nil, nil, nil, fmt.Errorf("get tenant: %w", err)
    }

    var cfg TenantConfig
    var metaBytes []byte
    if err := s.pool.QueryRow(ctx,
        `SELECT id, tenant_id, rule_set_id, workflow_template_id, metadata, version, created_at, updated_at
         FROM tenant_configs WHERE tenant_id = $1`,
        tenantID,
    ).Scan(&cfg.ID, &cfg.TenantID, &cfg.RuleSetId, &cfg.WorkflowTemplateID, &metaBytes, &cfg.Version, &cfg.CreatedAt, &cfg.UpdatedAt); err != nil {
        return nil, nil, nil, fmt.Errorf("get tenant config: %w", err)
    }
    if len(metaBytes) > 0 {
        _ = json.Unmarshal(metaBytes, &cfg.Metadata)
    }

    flags, err := s.getFlags(ctx, tenantID)
    if err != nil {
        return nil, nil, nil, err
    }
    return &t, &cfg, flags, nil
}

func (s *TenantStore) UpdateRuleConfig(ctx context.Context, tenantID, ruleSetID string) (*Tenant, *TenantConfig, []FeatureFlag, error) {
    var cfg TenantConfig
    var metaBytes []byte
    if err := s.pool.QueryRow(ctx,
        `UPDATE tenant_configs
         SET rule_set_id = $2, version = version + 1, updated_at = NOW()
         WHERE tenant_id = $1
         RETURNING id, tenant_id, rule_set_id, workflow_template_id, metadata, version, created_at, updated_at`,
        tenantID, ruleSetID,
    ).Scan(&cfg.ID, &cfg.TenantID, &cfg.RuleSetId, &cfg.WorkflowTemplateID, &metaBytes, &cfg.Version, &cfg.CreatedAt, &cfg.UpdatedAt); err != nil {
        return nil, nil, nil, fmt.Errorf("update rule config: %w", err)
    }
    if len(metaBytes) > 0 {
        _ = json.Unmarshal(metaBytes, &cfg.Metadata)
    }

    t, _, flags, err := s.GetTenant(ctx, tenantID)
    if err != nil {
        return nil, nil, nil, err
    }
    return t, &cfg, flags, nil
}

func (s *TenantStore) UpdateWorkflowConfig(ctx context.Context, tenantID, workflowTemplateID string) (*Tenant, *TenantConfig, []FeatureFlag, error) {
    var cfg TenantConfig
    var metaBytes []byte
    if err := s.pool.QueryRow(ctx,
        `UPDATE tenant_configs
         SET workflow_template_id = $2, version = version + 1, updated_at = NOW()
         WHERE tenant_id = $1
         RETURNING id, tenant_id, rule_set_id, workflow_template_id, metadata, version, created_at, updated_at`,
        tenantID, workflowTemplateID,
    ).Scan(&cfg.ID, &cfg.TenantID, &cfg.RuleSetId, &cfg.WorkflowTemplateID, &metaBytes, &cfg.Version, &cfg.CreatedAt, &cfg.UpdatedAt); err != nil {
        return nil, nil, nil, fmt.Errorf("update workflow config: %w", err)
    }
    if len(metaBytes) > 0 {
        _ = json.Unmarshal(metaBytes, &cfg.Metadata)
    }

    t, _, flags, err := s.GetTenant(ctx, tenantID)
    if err != nil {
        return nil, nil, nil, err
    }
    return t, &cfg, flags, nil
}

func (s *TenantStore) GetFeatureFlags(ctx context.Context, tenantID string) ([]FeatureFlag, error) {
    return s.getFlags(ctx, tenantID)
}

func (s *TenantStore) getFlags(ctx context.Context, tenantID string) ([]FeatureFlag, error) {
    rows, err := s.pool.Query(ctx,
        `SELECT id, tenant_id, key, enabled, rollout_percentage, created_at, updated_at
         FROM feature_flags WHERE tenant_id = $1`,
        tenantID,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to query feature flags: %w", err)
    }
    defer rows.Close()

    var flags []FeatureFlag
    for rows.Next() {
        var f FeatureFlag
        if err := rows.Scan(&f.ID, &f.TenantID, &f.Key, &f.Enabled, &f.RolloutPercentage, &f.CreatedAt, &f.UpdatedAt); err != nil {
            return nil, fmt.Errorf("failed to scan feature flag: %w", err)
        }
        flags = append(flags, f)
    }
    return flags, rows.Err()
}