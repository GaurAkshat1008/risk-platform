package cache

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

const rulesCacheTTL = 5 * time.Minute

// CachedRule is the Redis representation of an enabled rule.
type CachedRule struct {
    ID         string         `json:"id"`
    Name       string         `json:"name"`
    Version    int32          `json:"version"`
    Expression map[string]any `json:"expression"`
    Action     string         `json:"action"`
    Priority   int32          `json:"priority"`
}

type RuleCache struct {
    client *redis.Client
}

func NewRuleCache(addr string) (*RuleCache, error) {
    client := redis.NewClient(&redis.Options{Addr: addr})
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("failed to connect to Redis: %w", err)
    }
    return &RuleCache{client: client}, nil
}

// GetRules returns the cached enabled rules for a tenant. Returns nil, nil on cache miss.
func (c *RuleCache) GetRules(ctx context.Context, tenantID string) ([]CachedRule, error) {
    data, err := c.client.Get(ctx, rulesKey(tenantID)).Bytes()
    if errors.Is(err, redis.Nil) {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("redis get: %w", err)
    }
    var rules []CachedRule
    if err := json.Unmarshal(data, &rules); err != nil {
        return nil, fmt.Errorf("unmarshal rules: %w", err)
    }
    return rules, nil
}

// SetRules caches the enabled rules for a tenant.
func (c *RuleCache) SetRules(ctx context.Context, tenantID string, rules []CachedRule) error {
    data, err := json.Marshal(rules)
    if err != nil {
        return fmt.Errorf("marshal rules: %w", err)
    }
    return c.client.Set(ctx, rulesKey(tenantID), data, rulesCacheTTL).Err()
}

// Invalidate removes the cached rules for a tenant (call on any create/update/delete).
func (c *RuleCache) Invalidate(ctx context.Context, tenantID string) error {
    return c.client.Del(ctx, rulesKey(tenantID)).Err()
}

func (c *RuleCache) Close() error {
    return c.client.Close()
}

func rulesKey(tenantID string) string {
    return "rules-engine:rules:" + tenantID
}