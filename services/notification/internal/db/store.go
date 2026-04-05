package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

// Notification is the persisted notification record.
type Notification struct {
	ID            string
	TenantID      string
	Type          string
	Recipient     string
	Channel       string
	Payload       string
	Status        string
	Attempts      int
	LastAttemptAt *time.Time
	CreatedAt     time.Time
}

// NotificationPreference is the per-tenant, per-channel, per-event-type setting.
type NotificationPreference struct {
	TenantID  string
	Channel   string
	EventType string
	Enabled   bool
	Config    string // JSON config blob (webhook URL, email, Slack hook, etc.)
}

// NotificationStore provides all persistence for notifications and preferences.
type NotificationStore struct {
	pool *pgxpool.Pool
}

func NewNotificationStore(pool *pgxpool.Pool) *NotificationStore {
	return &NotificationStore{pool: pool}
}

// CreateNotification inserts a new notification with status=pending.
func (s *NotificationStore) CreateNotification(ctx context.Context, n *Notification) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO notifications
			(tenant_id, type, recipient, channel, payload, status, attempts)
		VALUES ($1, $2, $3, $4, $5, 'pending', 0)
		RETURNING id`,
		n.TenantID, n.Type, n.Recipient, n.Channel, n.Payload,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert notification: %w", err)
	}
	return id, nil
}

// GetNotification fetches a notification by id + tenant_id.
func (s *NotificationStore) GetNotification(ctx context.Context, id, tenantID string) (*Notification, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, type, recipient, channel, payload,
		       status, attempts, last_attempt_at, created_at
		FROM notifications
		WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	return scanNotification(row)
}

// ListPendingNotifications returns notifications ready for delivery (pending or retrying).
func (s *NotificationStore) ListPendingNotifications(ctx context.Context, limit int) ([]*Notification, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, type, recipient, channel, payload,
		       status, attempts, last_attempt_at, created_at
		FROM notifications
		WHERE status IN ('pending', 'retrying') AND attempts < 5
		ORDER BY created_at ASC
		LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list pending notifications: %w", err)
	}
	defer rows.Close()

	var out []*Notification
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// MarkDelivered updates a notification's status to delivered.
func (s *NotificationStore) MarkDelivered(ctx context.Context, id string) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx, `
		UPDATE notifications
		SET status = 'delivered', last_attempt_at = $2
		WHERE id = $1`,
		id, now,
	)
	return err
}

// MarkFailed increments attempts and sets status to retrying or failed (after 5 attempts).
func (s *NotificationStore) MarkFailed(ctx context.Context, id string) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx, `
		UPDATE notifications
		SET attempts        = attempts + 1,
		    last_attempt_at = $2,
		    status          = CASE WHEN attempts + 1 >= 5 THEN 'failed' ELSE 'retrying' END
		WHERE id = $1`,
		id, now,
	)
	return err
}

// UpsertPreference inserts or replaces a tenant notification preference.
func (s *NotificationStore) UpsertPreference(ctx context.Context, p *NotificationPreference) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notification_preferences
			(tenant_id, channel, event_type, enabled, config)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, channel, event_type) DO UPDATE
			SET enabled = EXCLUDED.enabled,
			    config  = EXCLUDED.config`,
		p.TenantID, p.Channel, p.EventType, p.Enabled, p.Config,
	)
	return err
}

// GetPreferences returns all preferences for a tenant.
func (s *NotificationStore) GetPreferences(ctx context.Context, tenantID string) ([]*NotificationPreference, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT tenant_id, channel, event_type, enabled, config
		FROM notification_preferences
		WHERE tenant_id = $1`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("query preferences: %w", err)
	}
	defer rows.Close()

	var out []*NotificationPreference
	for rows.Next() {
		var p NotificationPreference
		if err := rows.Scan(&p.TenantID, &p.Channel, &p.EventType, &p.Enabled, &p.Config); err != nil {
			return nil, fmt.Errorf("scan preference: %w", err)
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// -- helpers ------------------------------------------------------------------

type scanner interface {
	Scan(dest ...any) error
}

func scanNotification(row scanner) (*Notification, error) {
	var n Notification
	err := row.Scan(
		&n.ID, &n.TenantID, &n.Type, &n.Recipient, &n.Channel, &n.Payload,
		&n.Status, &n.Attempts, &n.LastAttemptAt, &n.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan notification: %w", err)
	}
	return &n, nil
}
