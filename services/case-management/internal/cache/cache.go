package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SLACache tracks SLA deadlines for cases in Redis.
// Key: cm:sla:{case_id}  Value: deadline RFC3339  TTL: time until deadline
// When the key expires, the SLA has been breached and the case should be escalated.
type SLACache struct {
	client *redis.Client
}

func NewSLACache(addr string) (*SLACache, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return &SLACache{client: client}, nil
}

// Register stores the SLA deadline for a case. TTL is set to time until deadline.
// If the deadline is already past, the key has no TTL (immediate breach detectable via DB).
func (c *SLACache) Register(ctx context.Context, caseID string, deadline time.Time) error {
	ttl := time.Until(deadline)
	if ttl <= 0 {
		// Already breached; don't store — DB query will catch it
		return nil
	}
	return c.client.Set(ctx, slaKey(caseID), deadline.UTC().Format(time.RFC3339), ttl).Err()
}

// IsTracked returns true if the case's SLA key is still present (not yet breached in Redis).
func (c *SLACache) IsTracked(ctx context.Context, caseID string) (bool, error) {
	n, err := c.client.Exists(ctx, slaKey(caseID)).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return n > 0, nil
}

// Remove deletes the SLA key (e.g. when a case is resolved or escalated).
func (c *SLACache) Remove(ctx context.Context, caseID string) error {
	return c.client.Del(ctx, slaKey(caseID)).Err()
}

func (c *SLACache) Close() error {
	return c.client.Close()
}

func slaKey(caseID string) string {
	return "cm:sla:" + caseID
}
