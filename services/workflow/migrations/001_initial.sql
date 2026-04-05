CREATE TABLE IF NOT EXISTS workflow_templates (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  TEXT        NOT NULL,
    name       TEXT        NOT NULL,
    version    INT         NOT NULL DEFAULT 1,
    states     TEXT[]      NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE TABLE IF NOT EXISTS workflow_transitions (
    id            UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id   UUID  NOT NULL REFERENCES workflow_templates(id) ON DELETE CASCADE,
    from_state    TEXT  NOT NULL,
    to_state      TEXT  NOT NULL,
    required_role TEXT  NOT NULL DEFAULT '',
    guards        JSONB NOT NULL DEFAULT '[]'
);

CREATE INDEX IF NOT EXISTS idx_workflow_templates_tenant ON workflow_templates (tenant_id);
CREATE INDEX IF NOT EXISTS idx_workflow_transitions_template ON workflow_transitions (template_id);
