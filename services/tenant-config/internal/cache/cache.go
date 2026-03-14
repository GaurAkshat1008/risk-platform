package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const tenantTTL = 5 * time.Minute

type FlagRecord struct {
	Key string `json:"key"`
	Enabled bool `json:"enabled"`
	RolloutPercentage int32 `json:"rollout_percentage"`
}

type TenantRecord struct {
	TenantID string `json:"tenant_id"`
	Name string `json:"name"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	RuleSetID string `json:"rule_set_id"`
	WorkflowTemplateID string `json:"workflow_template_id"`
	Metadata map[string]string `json:"metadata"`
	Version int32 `json:"version"`
	FeatureFlags []FlagRecord `json:"feature_flags"`
}

type TenantCache struct {
	client *redis.Client
}

func NewTenantCache(addr string) (*TenantCache, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	return &TenantCache{client: client}, nil
}

func (c *TenantCache) Set(ctx context.Context, rec TenantRecord) error {
    data, err := json.Marshal(rec)
    if err != nil {
        return fmt.Errorf("marshal: %w", err)
    }
    return c.client.Set(ctx, key(rec.TenantID), data, tenantTTL).Err()
}

func (c *TenantCache) Get(ctx context.Context, tenantID string) (*TenantRecord, error) {
    data, err := c.client.Get(ctx, key(tenantID)).Bytes()
    if errors.Is(err, redis.Nil) {
        return nil, nil // cache miss — not an error
    }
    if err != nil {
        return nil, fmt.Errorf("redis get: %w", err)
    }
    var rec TenantRecord
    if err := json.Unmarshal(data, &rec); err != nil {
        return nil, fmt.Errorf("unmarshal: %w", err)
    }
    return &rec, nil
}

func (c *TenantCache) Invalidate(ctx context.Context, tenantID string) error {
    return c.client.Del(ctx, key(tenantID)).Err()
}

func (c *TenantCache) Close() error {
    return c.client.Close()
}

func key(tenantID string) string {
    return "tenant-config:tenant:" + tenantID
}