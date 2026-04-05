package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

// LogEntry is the persisted representation of a single structured log line.
type LogEntry struct {
	ID          string
	Service     string
	Severity    string
	Message     string
	TraceID     string
	SpanID      string
	TenantID    string
	Environment string
	Attributes  []byte // raw JSON
	Timestamp   time.Time
}

// InsertParams carries the fields needed to store a new log entry.
type InsertParams struct {
	Service     string
	Severity    string
	Message     string
	TraceID     string
	SpanID      string
	TenantID    string
	Environment string
	Attributes  []byte
	Timestamp   time.Time
}

// QueryParams carries optional filters for querying log entries.
type QueryParams struct {
	Service         string
	Severity        string
	TraceID         string
	TenantID        string
	FromTime        time.Time
	ToTime          time.Time
	MessageContains string
	PageSize        int
	Offset          int
}

// LogStore provides persistence operations for log entries.
type LogStore struct {
	pool *pgxpool.Pool
}

func NewLogStore(pool *pgxpool.Pool) *LogStore {
	return &LogStore{pool: pool}
}

// Insert stores a new log entry and returns it with its generated ID and timestamp.
func (s *LogStore) Insert(ctx context.Context, p InsertParams) (*LogEntry, error) {
	if p.Attributes == nil {
		p.Attributes = []byte("{}")
	}
	ts := p.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	var e LogEntry
	err := s.pool.QueryRow(ctx,
		`INSERT INTO log_entries
		   (service, severity, message, trace_id, span_id, tenant_id, environment, attributes, timestamp)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, service, severity, message, trace_id, span_id,
		           tenant_id, environment, attributes, timestamp`,
		p.Service, p.Severity, p.Message, p.TraceID, p.SpanID,
		p.TenantID, p.Environment, p.Attributes, ts,
	).Scan(
		&e.ID, &e.Service, &e.Severity, &e.Message, &e.TraceID, &e.SpanID,
		&e.TenantID, &e.Environment, &e.Attributes, &e.Timestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("insert log entry: %w", err)
	}
	return &e, nil
}

// Query returns log entries matching the provided filters, newest first.
func (s *LogStore) Query(ctx context.Context, p QueryParams) ([]*LogEntry, error) {
	if p.PageSize <= 0 {
		p.PageSize = 50
	}
	if p.PageSize > 500 {
		p.PageSize = 500
	}

	args := []any{}
	conds := []string{}
	idx := 1

	if p.Service != "" {
		conds = append(conds, fmt.Sprintf("service = $%d", idx))
		args = append(args, p.Service)
		idx++
	}
	if p.Severity != "" {
		conds = append(conds, fmt.Sprintf("severity = $%d", idx))
		args = append(args, strings.ToUpper(p.Severity))
		idx++
	}
	if p.TraceID != "" {
		conds = append(conds, fmt.Sprintf("trace_id = $%d", idx))
		args = append(args, p.TraceID)
		idx++
	}
	if p.TenantID != "" {
		conds = append(conds, fmt.Sprintf("tenant_id = $%d", idx))
		args = append(args, p.TenantID)
		idx++
	}
	if !p.FromTime.IsZero() {
		conds = append(conds, fmt.Sprintf("timestamp >= $%d", idx))
		args = append(args, p.FromTime)
		idx++
	}
	if !p.ToTime.IsZero() {
		conds = append(conds, fmt.Sprintf("timestamp <= $%d", idx))
		args = append(args, p.ToTime)
		idx++
	}
	if p.MessageContains != "" {
		conds = append(conds, fmt.Sprintf("message ILIKE $%d", idx))
		args = append(args, "%"+p.MessageContains+"%")
		idx++
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	args = append(args, p.PageSize, p.Offset)
	query := fmt.Sprintf(
		`SELECT id, service, severity, message, trace_id, span_id,
		        tenant_id, environment, attributes, timestamp
		 FROM log_entries
		 %s
		 ORDER BY timestamp DESC
		 LIMIT $%d OFFSET $%d`,
		where, idx, idx+1,
	)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query log entries: %w", err)
	}
	defer rows.Close()

	var entries []*LogEntry
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(
			&e.ID, &e.Service, &e.Severity, &e.Message, &e.TraceID, &e.SpanID,
			&e.TenantID, &e.Environment, &e.Attributes, &e.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("scan log entry: %w", err)
		}
		entries = append(entries, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	// Suppress unused import warning — pgx.ErrNoRows used as sentinel elsewhere
	_ = pgx.ErrNoRows

	return entries, nil
}
