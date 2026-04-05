package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const dedupeTTL = 10 * time.Minute

type DedupeCache struct {
	client *redis.Client
}

func NewDedupeCache(addr string) (*DedupeCache, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Failed to connect to Redis: %w", err)
	}

	return &DedupeCache{client: client}, nil
}

func key(tenantID, idempotencyKey string) string {
	return "ingestion:dedupe:" + tenantID + ":" + idempotencyKey
}

func (c *DedupeCache) IsDuplicate(ctx context.Context, tenantID, idempotencyKey string) (bool, error) {
	n, err := c.client.Exists(ctx, key(tenantID, idempotencyKey)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return false, fmt.Errorf("Failed to check key in Redis: %w", err)
	}
	return n > 0, nil
}

func (c *DedupeCache) MarkProcessed(ctx context.Context, tenantID, idempotencyKey string) error {
	return c.client.Set(ctx, key(tenantID, idempotencyKey), "1", dedupeTTL).Err()
}

func (c *DedupeCache) Close() error {
	return c.client.Close()
}