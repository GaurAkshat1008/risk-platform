package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const rateLimitWindow = time.Minute

// RateLimiter enforces per-tenant, per-channel notification rate limits using Redis.
// Each key tracks the number of notifications sent in the current 1-minute window.
type RateLimiter struct {
	client *redis.Client
	limit  int64 // max notifications per tenant/channel per window
}

func NewRateLimiter(addr string, limit int64) (*RateLimiter, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return &RateLimiter{client: client, limit: limit}, nil
}

func (r *RateLimiter) Close() {
	_ = r.client.Close()
}

// Allow checks and increments the counter for tenantID+channel. Returns true if
// the notification is allowed (within the rate limit), false if it should be dropped.
func (r *RateLimiter) Allow(ctx context.Context, tenantID, channel string) (bool, error) {
	key := fmt.Sprintf("notif:rl:%s:%s", tenantID, channel)

	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, rateLimitWindow)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, fmt.Errorf("redis pipeline: %w", err)
	}

	return incr.Val() <= r.limit, nil
}
