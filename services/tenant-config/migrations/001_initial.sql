create extension if not exists "pgcrypto";

create table tenants (
  id uuid primary key default gen_random_uuid(),
  name text not null unique,
  status text not null default 'onboarding'
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
)

create table tenant_configs (
  id uuid primary key default gen_random_uuid(),
  tenant_id uui not null references tenants(id) on delete cascade,
  rule_set_id text not null default '',
  workfow_template_id text not null default '',
  metadata jsonb not null default '{}',
  version int not null default 1,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
)

create unique index idx_tenant_configs_tenant_id on tenant_configs(tenant_id);

create table feature_flags (
  id uuid primary key default gen_random_uuid(),
  tenant_id uuid not null references tenants(id) on delete cascade,
  key text not null,
  enabled boolean not null default false,
  rollout_percent int not null default 0,8 
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
)

create unique index idx_feature_flags_tenant_id on feature_flags(tenant_id);