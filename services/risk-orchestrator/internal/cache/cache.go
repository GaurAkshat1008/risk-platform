package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const inFlightTTL = 30 * time.Second

// InFlightCache prevents duplicate concurrent risk evaluations for the same payment event.
type InFlightCache struct {
	client *redis.Client
}

func NewInFlightCache(addr string) (*InFlightCache, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	return &InFlightCache{client: client}, nil
}

// TryAcquire attempts to mark a payment event as in-flight (SET NX).
// Returns true if the lock was acquired (proceed with evaluation).
// Returns false if another goroutine/instance is already processing it.
func (c *InFlightCache) TryAcquire(ctx context.Context, paymentEventID string) (bool, error) {
	ok, err := c.client.SetNX(ctx, inFlightKey(paymentEventID), "1", inFlightTTL).Result()
	if err != nil {
		return false, fmt.Errorf("redis setnx: %w", err)
	}
	return ok, nil
}

// Release removes the in-flight lock for a payment event.
func (c *InFlightCache) Release(ctx context.Context, paymentEventID string) error {
	return c.client.Del(ctx, inFlightKey(paymentEventID)).Err()
}

func (c *InFlightCache) Close() error {
	return c.client.Close()
}

func inFlightKey(paymentEventID string) string {
	return "risk-orchestrator:in-flight:" + paymentEventID
}
