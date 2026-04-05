CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS payment_events (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key TEXT        NOT NULL,
    tenant_id       UUID        NOT NULL,
    amount          BIGINT      NOT NULL,
    currency        TEXT        NOT NULL,
    source          TEXT        NOT NULL,
    destination     TEXT        NOT NULL,
    metadata        JSONB       NOT NULL DEFAULT '{}',
    status          TEXT        NOT NULL DEFAULT 'received',
    received_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_payment_events_tenant_id   ON payment_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_payment_events_received_at ON payment_events(received_at);