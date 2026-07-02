-- PES enterprise metadata store baseline.
-- Production source of truth is Postgres. Local/dev adapters may exist, but the
-- platform model is defined by this schema: tenant/project scoped resources,
-- transactional current-state + append-only events, durable queue leases,
-- attempts, results, artifacts, audit, policy, and secrets.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS tenants (
  id text PRIMARY KEY,
  name text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS projects (
  id text NOT NULL,
  tenant_id text NOT NULL REFERENCES tenants(id),
  name text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, id)
);

CREATE TABLE IF NOT EXISTS users (
  id text PRIMARY KEY,
  tenant_id text NOT NULL,
  project_id text NOT NULL,
  username text NOT NULL,
  role text NOT NULL CHECK (role IN ('user','admin','super_admin')),
  token_hash text NOT NULL,
  password_hash text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, username)
);

CREATE TABLE IF NOT EXISTS plugins (
  id text PRIMARY KEY,
  tenant_id text NOT NULL,
  project_id text NOT NULL,
  name text NOT NULL,
  version text NOT NULL,
  api_version text NOT NULL,
  runtime_type text NOT NULL,
  protocol text NOT NULL,
  command text,
  args_json jsonb NOT NULL DEFAULT '[]'::jsonb,
  image text,
  work_dir text,
  capabilities jsonb NOT NULL DEFAULT '[]'::jsonb,
  permissions jsonb NOT NULL DEFAULT '{}'::jsonb,
  resources jsonb NOT NULL DEFAULT '{}'::jsonb,
  secret_refs jsonb NOT NULL DEFAULT '{}'::jsonb,
  checksum text NOT NULL,
  signature text,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  status text NOT NULL,
  error_message text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, project_id, name, version)
);


CREATE TABLE IF NOT EXISTS plugin_registry (
  plugin_id text PRIMARY KEY REFERENCES plugins(id),
  name text NOT NULL,
  version text NOT NULL,
  source_path text NOT NULL,
  manifest_hash text NOT NULL,
  last_seen_at timestamptz NOT NULL,
  synced_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS plugin_registry_name_version_idx ON plugin_registry (name, version);

CREATE TABLE IF NOT EXISTS executions (
  id text PRIMARY KEY,
  tenant_id text NOT NULL,
  project_id text NOT NULL,
  user_id text NOT NULL,
  plugin_ids_json jsonb NOT NULL,
  input_json jsonb NOT NULL,
  input_hash text NOT NULL,
  plugin_ids_hash text NOT NULL,
  idempotency_key text NOT NULL DEFAULT '',
  status text NOT NULL,
  error_message text,
  created_at timestamptz NOT NULL DEFAULT now(),
  queued_at timestamptz,
  started_at timestamptz,
  finished_at timestamptz,
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS executions_idempotency_key_uidx
  ON executions (tenant_id, project_id, user_id, idempotency_key)
  WHERE idempotency_key <> '';
CREATE INDEX IF NOT EXISTS executions_scope_status_idx ON executions (tenant_id, project_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS task_events (
  id text PRIMARY KEY,
  tenant_id text NOT NULL,
  project_id text NOT NULL,
  execution_id text NOT NULL REFERENCES executions(id),
  plugin_id text,
  type text NOT NULL,
  status text,
  message text,
  detail_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  request_id text,
  trace_id text,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS task_events_execution_idx ON task_events (execution_id, created_at ASC);

CREATE TABLE IF NOT EXISTS task_attempts (
  id text PRIMARY KEY,
  tenant_id text NOT NULL,
  project_id text NOT NULL,
  execution_id text NOT NULL REFERENCES executions(id),
  attempt_no integer NOT NULL,
  worker_id text NOT NULL,
  lease_id text,
  status text NOT NULL,
  heartbeat_at timestamptz,
  lease_until timestamptz,
  started_at timestamptz NOT NULL DEFAULT now(),
  finished_at timestamptz,
  error_message text,
  UNIQUE (execution_id, attempt_no)
);

CREATE TABLE IF NOT EXISTS task_queue (
  id bigserial PRIMARY KEY,
  tenant_id text NOT NULL,
  project_id text NOT NULL,
  task_id text NOT NULL REFERENCES executions(id),
  status text NOT NULL CHECK (status IN ('ready','leased','acked','dlq')),
  attempt_no integer NOT NULL DEFAULT 0,
  available_at timestamptz NOT NULL DEFAULT now(),
  lease_id text NOT NULL DEFAULT '',
  lease_until timestamptz,
  worker_id text NOT NULL DEFAULT '',
  heartbeat_at timestamptz,
  last_error text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, project_id, task_id)
);
CREATE INDEX IF NOT EXISTS task_queue_ready_idx ON task_queue (status, available_at, created_at);
CREATE INDEX IF NOT EXISTS task_queue_lease_idx ON task_queue (status, lease_until);

CREATE TABLE IF NOT EXISTS execution_results (
  id text PRIMARY KEY,
  tenant_id text NOT NULL,
  project_id text NOT NULL,
  execution_id text NOT NULL REFERENCES executions(id),
  plugin_id text NOT NULL,
  plugin_name text NOT NULL,
  status text NOT NULL,
  output_json jsonb,
  output_ref text,
  result_hash text,
  error_message text,
  stdout_preview text,
  stderr_preview text,
  exit_code integer,
  duration_ms bigint,
  started_at timestamptz,
  finished_at timestamptz
);
CREATE INDEX IF NOT EXISTS execution_results_execution_idx ON execution_results (execution_id);

CREATE TABLE IF NOT EXISTS artifacts (
  id text PRIMARY KEY,
  tenant_id text NOT NULL,
  project_id text NOT NULL,
  execution_id text NOT NULL REFERENCES executions(id),
  result_id text,
  kind text NOT NULL,
  object_uri text NOT NULL,
  digest text NOT NULL,
  size_bytes bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id text PRIMARY KEY,
  tenant_id text NOT NULL,
  project_id text NOT NULL,
  actor_id text NOT NULL,
  actor_type text NOT NULL DEFAULT 'user',
  action text NOT NULL,
  resource_type text NOT NULL,
  resource_id text NOT NULL,
  decision text NOT NULL CHECK (decision IN ('allow','deny','error')),
  reason text,
  request_id text,
  trace_id text,
  plugin_digest text,
  input_hash text,
  result_hash text,
  message text NOT NULL,
  detail_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS audit_scope_time_idx ON audit_logs (tenant_id, project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_resource_idx ON audit_logs (resource_type, resource_id, created_at ASC);

CREATE TABLE IF NOT EXISTS secrets (
  id text PRIMARY KEY,
  tenant_id text NOT NULL,
  project_id text NOT NULL,
  name text NOT NULL,
  provider text NOT NULL,
  external_ref text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, project_id, name)
);

CREATE TABLE IF NOT EXISTS policies (
  id text PRIMARY KEY,
  tenant_id text NOT NULL,
  project_id text NOT NULL,
  name text NOT NULL,
  engine text NOT NULL DEFAULT 'rego',
  source text NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now()
);


CREATE TABLE IF NOT EXISTS webhook_endpoints (
  id text PRIMARY KEY,
  tenant_id text NOT NULL,
  project_id text NOT NULL,
  name text NOT NULL,
  url text NOT NULL,
  secret text NOT NULL,
  events_json jsonb NOT NULL DEFAULT '[]'::jsonb,
  status text NOT NULL CHECK (status IN ('enabled','disabled')),
  created_by text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS webhook_endpoints_scope_idx ON webhook_endpoints (tenant_id, project_id, status);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
  id text PRIMARY KEY,
  tenant_id text NOT NULL,
  project_id text NOT NULL,
  webhook_id text NOT NULL REFERENCES webhook_endpoints(id),
  event_id text NOT NULL,
  event_type text NOT NULL,
  target_url text NOT NULL,
  status text NOT NULL CHECK (status IN ('pending','delivered','failed','dlq')),
  attempt_no integer NOT NULL DEFAULT 1,
  max_attempts integer NOT NULL DEFAULT 5,
  next_retry_at timestamptz,
  status_code integer,
  error text,
  payload_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  delivered_at timestamptz
);
CREATE INDEX IF NOT EXISTS webhook_deliveries_webhook_idx ON webhook_deliveries (webhook_id, created_at DESC);

CREATE INDEX IF NOT EXISTS webhook_deliveries_retry_idx ON webhook_deliveries (status, next_retry_at) WHERE status = 'failed';
