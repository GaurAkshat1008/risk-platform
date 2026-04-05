CREATE TABLE IF NOT EXISTS cases (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_id      TEXT        NOT NULL,
    tenant_id        TEXT        NOT NULL,
    assignee_id      TEXT        NOT NULL DEFAULT '',
    status           TEXT        NOT NULL DEFAULT 'open'
                                 CHECK (status IN ('open', 'in_review', 'resolved', 'escalated')),
    priority         TEXT        NOT NULL DEFAULT 'medium'
                                 CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    payment_event_id TEXT        NOT NULL DEFAULT '',
    outcome          TEXT        NOT NULL DEFAULT '',
    sla_deadline     TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (decision_id, tenant_id)
);

CREATE TABLE IF NOT EXISTS case_actions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id    UUID        NOT NULL REFERENCES cases(id),
    actor_id   TEXT        NOT NULL,
    action     TEXT        NOT NULL,
    notes      TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cases_tenant_id    ON cases (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cases_status       ON cases (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_cases_assignee     ON cases (assignee_id);
CREATE INDEX IF NOT EXISTS idx_cases_sla_deadline ON cases (sla_deadline) WHERE status IN ('open', 'in_review');
CREATE INDEX IF NOT EXISTS idx_case_actions_case  ON case_actions (case_id);
