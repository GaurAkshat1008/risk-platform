package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultTTL = 30 * time.Second

// QueryCache is a Redis-backed result cache for GraphQL queries.
// Keys are tenant-scoped to prevent cross-tenant data leaks.
type QueryCache struct {
	client *redis.Client
}

func NewQueryCache(addr string) (*QueryCache, error) {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &QueryCache{client: rdb}, nil
}

func (c *QueryCache) Close() error {
	return c.client.Close()
}

// Get retrieves a cached value by query name + args. Returns false if not cached.
func (c *QueryCache) Get(ctx context.Context, tenantID, queryName string, args any, dest any) (bool, error) {
	key, err := buildKey(tenantID, queryName, args)
	if err != nil {
		return false, nil // non-fatal: proceed without cache
	}

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, nil // non-fatal on Redis errors
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return false, nil
	}
	return true, nil
}

// Set stores a value in the cache with the default TTL.
func (c *QueryCache) Set(ctx context.Context, tenantID, queryName string, args any, value any) {
	c.SetWithTTL(ctx, tenantID, queryName, args, value, defaultTTL)
}

// SetWithTTL stores a value with a custom TTL.
func (c *QueryCache) SetWithTTL(ctx context.Context, tenantID, queryName string, args any, value any, ttl time.Duration) {
	key, err := buildKey(tenantID, queryName, args)
	if err != nil {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = c.client.Set(ctx, key, data, ttl).Err()
}

// Invalidate removes a single cached entry.
func (c *QueryCache) Invalidate(ctx context.Context, tenantID, queryName string, args any) {
	key, _ := buildKey(tenantID, queryName, args)
	_ = c.client.Del(ctx, key).Err()
}

// InvalidatePrefix removes all keys matching bff:{tenantID}:{prefix}*.
func (c *QueryCache) InvalidatePrefix(ctx context.Context, tenantID, prefix string) {
	pattern := fmt.Sprintf("bff:%s:%s*", tenantID, prefix)
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil || len(keys) == 0 {
		return
	}
	_ = c.client.Del(ctx, keys...).Err()
}

func buildKey(tenantID, queryName string, args any) (string, error) {
	b, err := json.Marshal(args)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return fmt.Sprintf("bff:%s:%s:%x", tenantID, queryName, h[:8]), nil
}
