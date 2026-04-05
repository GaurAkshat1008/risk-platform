-- Append-only audit event table with hash chain for tamper detection.
-- No UPDATE or DELETE operations should ever be issued against this table.
CREATE TABLE IF NOT EXISTS audit_events (
    seq           BIGSERIAL   NOT NULL,
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     TEXT        NOT NULL,
    actor_id      TEXT        NOT NULL DEFAULT '',
    action        TEXT        NOT NULL,
    resource_type TEXT        NOT NULL,
    resource_id   TEXT        NOT NULL DEFAULT '',
    source_topic  TEXT        NOT NULL DEFAULT '',
    payload       BYTEA       NOT NULL DEFAULT '',
    previous_hash TEXT        NOT NULL DEFAULT '',
    hash          TEXT        NOT NULL,
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_events_seq        ON audit_events (seq);
CREATE INDEX       IF NOT EXISTS idx_audit_events_tenant      ON audit_events (tenant_id, seq ASC);
CREATE INDEX       IF NOT EXISTS idx_audit_events_resource    ON audit_events (tenant_id, resource_type, resource_id);
CREATE INDEX       IF NOT EXISTS idx_audit_events_occurred_at ON audit_events (tenant_id, occurred_at DESC);
CREATE INDEX       IF NOT EXISTS idx_audit_events_actor       ON audit_events (tenant_id, actor_id);
