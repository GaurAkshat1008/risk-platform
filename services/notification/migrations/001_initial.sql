-- Notification Service: initial schema

-- Outbound notification log.
CREATE TABLE IF NOT EXISTS notifications (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       TEXT        NOT NULL,
    type            TEXT        NOT NULL,           -- e.g. "case.created", "decision.made"
    recipient       TEXT        NOT NULL,           -- email addr, webhook URL, Slack channel
    channel         TEXT        NOT NULL DEFAULT 'webhook'
                                CHECK (channel IN ('email', 'webhook', 'slack')),
    payload         TEXT        NOT NULL DEFAULT '',
    status          TEXT        NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending', 'delivered', 'failed', 'retrying')),
    attempts        INT         NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_tenant_id  ON notifications (tenant_id);
CREATE INDEX IF NOT EXISTS idx_notifications_status     ON notifications (status) WHERE status IN ('pending', 'retrying');
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications (created_at DESC);

-- Per-tenant notification preference table.
-- UNIQUE on (tenant_id, channel, event_type) so preferences can be upserted cleanly.
CREATE TABLE IF NOT EXISTS notification_preferences (
    tenant_id   TEXT    NOT NULL,
    channel     TEXT    NOT NULL CHECK (channel IN ('email', 'webhook', 'slack')),
    event_type  TEXT    NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    config      TEXT    NOT NULL DEFAULT '{}',   -- JSON config blob

    CONSTRAINT notification_preferences_pkey PRIMARY KEY (tenant_id, channel, event_type)
);

CREATE INDEX IF NOT EXISTS idx_notification_prefs_tenant ON notification_preferences (tenant_id);
