-- +migrate Up
ALTER TABLE users
  ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN last_login_at TIMESTAMPTZ,
  ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Keep the MVP simple now, but model memberships explicitly so tenant-scoped RBAC
-- can grow later without rewriting user records again.
CREATE TABLE tenant_memberships (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  role TEXT NOT NULL CHECK (role IN ('admin', 'viewer')),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  invited_by_user_id TEXT REFERENCES users (id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  disabled_at TIMESTAMPTZ,
  UNIQUE (tenant_id, user_id)
);

CREATE INDEX tenant_memberships_user_idx
  ON tenant_memberships (user_id);

CREATE INDEX tenant_memberships_tenant_role_idx
  ON tenant_memberships (tenant_id, role);

INSERT INTO tenant_memberships (id, tenant_id, user_id, role, is_active, invited_by_user_id, created_at, updated_at, disabled_at)
SELECT
  'tm-' || u.id,
  u.tenant_id,
  u.id,
  CASE
    WHEN u.role IN ('owner', 'admin') THEN 'admin'
    ELSE 'viewer'
  END,
  TRUE,
  NULL,
  u.created_at,
  NOW(),
  NULL
FROM users u
ON CONFLICT (tenant_id, user_id) DO NOTHING;

CREATE TABLE user_sessions (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  session_token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at TIMESTAMPTZ,
  user_agent TEXT NOT NULL DEFAULT '',
  ip_address TEXT NOT NULL DEFAULT ''
);

CREATE INDEX user_sessions_user_idx
  ON user_sessions (user_id, expires_at DESC);

CREATE INDEX user_sessions_tenant_idx
  ON user_sessions (tenant_id, expires_at DESC);

CREATE TABLE user_invites (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  email TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('viewer')),
  invite_token_hash TEXT NOT NULL UNIQUE,
  invited_by_user_id TEXT REFERENCES users (id) ON DELETE SET NULL,
  accepted_by_user_id TEXT REFERENCES users (id) ON DELETE SET NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  accepted_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX user_invites_pending_email_idx
  ON user_invites (tenant_id, lower(email))
  WHERE accepted_at IS NULL AND revoked_at IS NULL;

ALTER TABLE servers
  ADD COLUMN cpu_threads INT NOT NULL DEFAULT 0,
  ADD COLUMN total_memory_gb DOUBLE PRECISION NOT NULL DEFAULT 0,
  ADD COLUMN total_disk_gb DOUBLE PRECISION NOT NULL DEFAULT 0,
  ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE service_groups
  ADD COLUMN last_log_timestamp TIMESTAMPTZ,
  ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX service_groups_server_compose_idx
  ON service_groups (server_id, compose_project);

ALTER TABLE containers
  ADD COLUMN network_mb DOUBLE PRECISION NOT NULL DEFAULT 0,
  ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX containers_service_group_name_idx
  ON containers (service_group_id, name);

ALTER TABLE metric_points
  ADD COLUMN container_id TEXT REFERENCES containers (id) ON DELETE CASCADE;

CREATE INDEX metric_points_container_key_recorded_at_idx
  ON metric_points (container_id, metric_key, recorded_at DESC);

ALTER TABLE log_lines
  ADD COLUMN container_name TEXT NOT NULL DEFAULT '',
  ADD COLUMN service_tag TEXT NOT NULL DEFAULT '',
  ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX log_lines_container_lookup_idx
  ON log_lines (container_id, occurred_at DESC);

-- +migrate Down
DROP INDEX IF EXISTS log_lines_container_lookup_idx;

ALTER TABLE log_lines
  DROP COLUMN IF EXISTS created_at,
  DROP COLUMN IF EXISTS service_tag,
  DROP COLUMN IF EXISTS container_name;

DROP INDEX IF EXISTS metric_points_container_key_recorded_at_idx;

ALTER TABLE metric_points
  DROP COLUMN IF EXISTS container_id;

DROP INDEX IF EXISTS containers_service_group_name_idx;

ALTER TABLE containers
  DROP COLUMN IF EXISTS created_at,
  DROP COLUMN IF EXISTS network_mb;

DROP INDEX IF EXISTS service_groups_server_compose_idx;

ALTER TABLE service_groups
  DROP COLUMN IF EXISTS created_at,
  DROP COLUMN IF EXISTS last_log_timestamp;

ALTER TABLE servers
  DROP COLUMN IF EXISTS updated_at,
  DROP COLUMN IF EXISTS total_disk_gb,
  DROP COLUMN IF EXISTS total_memory_gb,
  DROP COLUMN IF EXISTS cpu_threads;

DROP INDEX IF EXISTS user_invites_pending_email_idx;
DROP TABLE IF EXISTS user_invites;

DROP INDEX IF EXISTS user_sessions_tenant_idx;
DROP INDEX IF EXISTS user_sessions_user_idx;
DROP TABLE IF EXISTS user_sessions;

DROP INDEX IF EXISTS tenant_memberships_tenant_role_idx;
DROP INDEX IF EXISTS tenant_memberships_user_idx;
DROP TABLE IF EXISTS tenant_memberships;

ALTER TABLE users
  DROP COLUMN IF EXISTS updated_at,
  DROP COLUMN IF EXISTS last_login_at,
  DROP COLUMN IF EXISTS is_active;
