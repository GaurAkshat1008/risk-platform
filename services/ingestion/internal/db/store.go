package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentStatus string

const (
	PaymentStatusReceived PaymentStatus = "received"
	PaymentStatusRejected PaymentStatus = "rejected"
)

type PaymentEvent struct {
	ID string
	IdempotencyKey string
	TenantID string
	Amount int64
	Currency string
	Source string
	Destination string
	Metadata map[string]string
	Status PaymentStatus
	ReceivedAt time.Time
}

type PaymentStore struct {
	pool *pgxpool.Pool
}

func NewPaymentStore(pool *pgxpool.Pool) *PaymentStore {
	return &PaymentStore{pool: pool}
}

func (s *PaymentStore) InsertPaymentEvent(ctx context.Context, event *PaymentEvent) (*PaymentEvent, bool, error) {
	metaJson, err := json.Marshal(event.Metadata)
	if err != nil {
		return nil, false, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var result PaymentEvent
	var metaDataBytes []byte

	err = s.pool.QueryRow(ctx, `
		INSERT INTO payment_events (id, idempotency_key, tenant_id, amount, currency, source, destination, metadata, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_id, idempotency_key) DO NOTHING
		RETURNING id, idempotency_key, tenant_id, amount, currency, source, destination, metadata, status, created_at
	`, event.ID, event.IdempotencyKey, event.TenantID, event.Amount, event.Currency, event.Source, event.Destination, metaJson, event.Status).Scan(
		&result.ID, &result.IdempotencyKey, &result.TenantID, &result.Amount, &result.Currency, &result.Source, &result.Destination, &metaDataBytes, &result.Status, &result.ReceivedAt)

		if errors.Is(err, pgx.ErrNoRows) {
			existing, fetchErr := s.GetPaymentEventByIdempotencyKey(ctx, event.TenantID, event.IdempotencyKey)
			if fetchErr != nil {
				return nil, false, fmt.Errorf("failed to fetch existing payment event: %w", fetchErr)
			}
			return existing, false, nil
		}
		if err != nil {
			return nil, false, fmt.Errorf("insert payment_event: %w", err)
		}
		if len(metaDataBytes) > 0 {
			_ = json.Unmarshal(metaDataBytes, &result.Metadata)
		}
		return &result, true, nil
}

func (s *PaymentStore) GetPaymentEventByIdempotencyKey(ctx context.Context, tenantID, idempotencyKey string) (*PaymentEvent, error) {
	var result PaymentEvent
	var metaDataBytes []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id, idempotency_key, tenant_id, amount, currency, source, destination, metadata, status, created_at
		FROM payment_events
		WHERE tenant_id = $1 AND idempotency_key = $2
	`, tenantID, idempotencyKey).Scan(
		&result.ID, &result.IdempotencyKey, &result.TenantID, &result.Amount, &result.Currency, &result.Source, &result.Destination, &metaDataBytes, &result.Status, &result.ReceivedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query payment_event by idempotency key: %w", err)
	}
	if len(metaDataBytes) > 0 {
		_ = json.Unmarshal(metaDataBytes, &result.Metadata)
	}
	return &result, nil
}
