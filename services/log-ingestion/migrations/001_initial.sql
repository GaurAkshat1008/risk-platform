-- Append-only structured log storage table.
-- Entries are written once and never updated or deleted — only queried.
CREATE TABLE IF NOT EXISTS log_entries (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    service     TEXT        NOT NULL,
    severity    TEXT        NOT NULL CHECK (severity IN ('DEBUG','INFO','WARN','ERROR','FATAL')),
    message     TEXT        NOT NULL DEFAULT '',
    trace_id    TEXT        NOT NULL DEFAULT '',
    span_id     TEXT        NOT NULL DEFAULT '',
    tenant_id   TEXT        NOT NULL DEFAULT '',
    environment TEXT        NOT NULL DEFAULT '',
    attributes  JSONB       NOT NULL DEFAULT '{}',
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_log_service   ON log_entries (service, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_log_severity  ON log_entries (severity, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_log_trace_id  ON log_entries (trace_id) WHERE trace_id != '';
CREATE INDEX IF NOT EXISTS idx_log_tenant    ON log_entries (tenant_id, timestamp DESC) WHERE tenant_id != '';
CREATE INDEX IF NOT EXISTS idx_log_timestamp ON log_entries (timestamp DESC);
