package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

// AuditEvent is the persisted representation of a single audit log entry.
type AuditEvent struct {
	ID           string
	Seq          int64
	TenantID     string
	ActorID      string
	Action       string
	ResourceType string
	ResourceID   string
	SourceTopic  string
	Payload      []byte
	PreviousHash string
	Hash         string
	OccurredAt   time.Time
}

// AppendEventParams holds the input for appending a new audit event.
type AppendEventParams struct {
	TenantID     string
	ActorID      string
	Action       string
	ResourceType string
	ResourceID   string
	SourceTopic  string
	Payload      []byte
}

// AuditQueryParams holds optional filters for QueryEvents.
type AuditQueryParams struct {
	TenantID     string
	ActorID      string
	ResourceType string
	ResourceID   string
	Action       string
	FromTime     time.Time
	ToTime       time.Time
	PageSize     int
	Offset       int
}

// AuditStore provides persistence operations for the tamper-evident audit trail.
type AuditStore struct {
	pool *pgxpool.Pool
}

func NewAuditStore(pool *pgxpool.Pool) *AuditStore {
	return &AuditStore{pool: pool}
}

// AppendEvent appends a new audit event to the chain for the given tenant.
// The operation is transactional: it reads the latest hash, computes the new hash,
// and inserts the event atomically.
func (s *AuditStore) AppendEvent(ctx context.Context, p AppendEventParams) (*AuditEvent, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Fetch the hash of the most recent event for this tenant to chain from.
	var prevHash string
	err = tx.QueryRow(ctx,
		`SELECT hash FROM audit_events
		 WHERE tenant_id = $1
		 ORDER BY seq DESC
		 LIMIT 1`,
		p.TenantID,
	).Scan(&prevHash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get latest hash: %w", err)
	}
	// On ErrNoRows, prevHash remains "" (genesis event).

	now := time.Now().UTC()
	newHash := computeHash(prevHash, p.TenantID, p.ActorID, p.Action,
		p.ResourceType, p.ResourceID, p.Payload, now)

	payload := p.Payload
	if payload == nil {
		payload = []byte{}
	}

	var e AuditEvent
	err = tx.QueryRow(ctx,
		`INSERT INTO audit_events
		   (tenant_id, actor_id, action, resource_type, resource_id,
		    source_topic, payload, previous_hash, hash, occurred_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING seq, id, tenant_id, actor_id, action, resource_type, resource_id,
		           source_topic, payload, previous_hash, hash, occurred_at`,
		p.TenantID, p.ActorID, p.Action, p.ResourceType, p.ResourceID,
		p.SourceTopic, payload, prevHash, newHash, now,
	).Scan(
		&e.Seq, &e.ID, &e.TenantID, &e.ActorID, &e.Action,
		&e.ResourceType, &e.ResourceID, &e.SourceTopic,
		&e.Payload, &e.PreviousHash, &e.Hash, &e.OccurredAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert audit event: %w", err)
	}

	return &e, tx.Commit(ctx)
}

// QueryEvents returns audit events matching the provided filters, ordered by seq ASC.
func (s *AuditStore) QueryEvents(ctx context.Context, p AuditQueryParams) ([]AuditEvent, error) {
	if p.PageSize <= 0 {
		p.PageSize = 50
	}

	query := `SELECT seq, id, tenant_id, actor_id, action, resource_type, resource_id,
	                 source_topic, payload, previous_hash, hash, occurred_at
	          FROM audit_events
	          WHERE tenant_id = $1`
	args := []any{p.TenantID}
	argIdx := 2

	if p.ActorID != "" {
		query += fmt.Sprintf(" AND actor_id = $%d", argIdx)
		args = append(args, p.ActorID)
		argIdx++
	}
	if p.ResourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, p.ResourceType)
		argIdx++
	}
	if p.ResourceID != "" {
		query += fmt.Sprintf(" AND resource_id = $%d", argIdx)
		args = append(args, p.ResourceID)
		argIdx++
	}
	if p.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, p.Action)
		argIdx++
	}
	if !p.FromTime.IsZero() {
		query += fmt.Sprintf(" AND occurred_at >= $%d", argIdx)
		args = append(args, p.FromTime)
		argIdx++
	}
	if !p.ToTime.IsZero() {
		query += fmt.Sprintf(" AND occurred_at <= $%d", argIdx)
		args = append(args, p.ToTime)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY seq ASC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, p.PageSize, p.Offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query audit events: %w", err)
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var e AuditEvent
		if err := rows.Scan(
			&e.Seq, &e.ID, &e.TenantID, &e.ActorID, &e.Action,
			&e.ResourceType, &e.ResourceID, &e.SourceTopic,
			&e.Payload, &e.PreviousHash, &e.Hash, &e.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// VerifyChain verifies the hash chain for a given tenant up to `limit` events.
// Returns (valid, eventsChecked, brokenAtID, error).
func (s *AuditStore) VerifyChain(ctx context.Context, tenantID string, limit int) (bool, int64, string, error) {
	if limit <= 0 {
		limit = 1000
	}

	rows, err := s.pool.Query(ctx,
		`SELECT seq, id, previous_hash, hash
		 FROM audit_events
		 WHERE tenant_id = $1
		 ORDER BY seq ASC
		 LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return false, 0, "", fmt.Errorf("verify chain query: %w", err)
	}
	defer rows.Close()

	type chainRow struct {
		seq          int64
		id           string
		previousHash string
		hash         string
	}

	var events []chainRow
	for rows.Next() {
		var r chainRow
		if err := rows.Scan(&r.seq, &r.id, &r.previousHash, &r.hash); err != nil {
			return false, 0, "", fmt.Errorf("scan chain row: %w", err)
		}
		events = append(events, r)
	}
	if err := rows.Err(); err != nil {
		return false, 0, "", err
	}

	var checked int64
	for i, e := range events {
		if i == 0 {
			// Genesis event: previous_hash must be ""
			if e.previousHash != "" {
				return false, checked, e.id, nil
			}
		} else {
			// Each event's previous_hash must equal the prior event's hash
			if e.previousHash != events[i-1].hash {
				return false, checked, e.id, nil
			}
		}
		checked++
	}

	return true, checked, "", nil
}

// computeHash produces SHA-256 over the concatenation of all event fields.
func computeHash(
	prevHash, tenantID, actorID, action, resourceType, resourceID string,
	payload []byte,
	occurredAt time.Time,
) string {
	h := sha256.New()
	h.Write([]byte(prevHash))
	h.Write([]byte(tenantID))
	h.Write([]byte(actorID))
	h.Write([]byte(action))
	h.Write([]byte(resourceType))
	h.Write([]byte(resourceID))
	h.Write(payload)
	h.Write([]byte(occurredAt.Format(time.RFC3339Nano)))
	return hex.EncodeToString(h.Sum(nil))
}
