package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"workflow/internal/db"

	"github.com/redis/go-redis/v9"
)

const ttl = 5 * time.Minute

// WorkflowCache caches workflow templates by (template_id, tenant_id).
type WorkflowCache struct {
	client *redis.Client
}

func NewWorkflowCache(addr string) (*WorkflowCache, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return &WorkflowCache{client: client}, nil
}

// Get returns a cached template or (nil, nil) on a cache miss.
func (c *WorkflowCache) Get(ctx context.Context, templateID, tenantID string) (*db.WorkflowTemplate, error) {
	data, err := c.client.Get(ctx, cacheKey(templateID, tenantID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}
	var tmpl db.WorkflowTemplate
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("unmarshal cached template: %w", err)
	}
	return &tmpl, nil
}

// Set stores a template in the cache.
func (c *WorkflowCache) Set(ctx context.Context, tmpl *db.WorkflowTemplate) error {
	data, err := json.Marshal(tmpl)
	if err != nil {
		return fmt.Errorf("marshal template: %w", err)
	}
	return c.client.Set(ctx, cacheKey(tmpl.ID, tmpl.TenantID), data, ttl).Err()
}

// Invalidate removes a template from the cache.
func (c *WorkflowCache) Invalidate(ctx context.Context, templateID, tenantID string) error {
	return c.client.Del(ctx, cacheKey(templateID, tenantID)).Err()
}

func (c *WorkflowCache) Close() error {
	return c.client.Close()
}

func cacheKey(templateID, tenantID string) string {
	return "wf:tmpl:" + tenantID + ":" + templateID
}
