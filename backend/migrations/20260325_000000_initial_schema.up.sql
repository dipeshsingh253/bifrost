CREATE TABLE tenants (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE users (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  name TEXT NOT NULL,
  role TEXT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, email)
);

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

CREATE TABLE servers (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  hostname TEXT NOT NULL,
  public_ip TEXT NOT NULL,
  agent_version TEXT NOT NULL,
  status TEXT NOT NULL,
  last_seen_at TIMESTAMPTZ,
  uptime_seconds BIGINT NOT NULL DEFAULT 0,
  cpu_usage_pct DOUBLE PRECISION NOT NULL DEFAULT 0,
  memory_usage_pct DOUBLE PRECISION NOT NULL DEFAULT 0,
  disk_usage_pct DOUBLE PRECISION NOT NULL DEFAULT 0,
  network_rx_mb DOUBLE PRECISION NOT NULL DEFAULT 0,
  network_tx_mb DOUBLE PRECISION NOT NULL DEFAULT 0,
  load_average TEXT NOT NULL DEFAULT '',
  os TEXT NOT NULL DEFAULT '',
  kernel TEXT NOT NULL DEFAULT '',
  cpu_model TEXT NOT NULL DEFAULT '',
  cpu_threads INT NOT NULL DEFAULT 0,
  cpu_cores INT NOT NULL DEFAULT 0,
  total_memory_gb DOUBLE PRECISION NOT NULL DEFAULT 0,
  total_disk_gb DOUBLE PRECISION NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE service_groups (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  server_id TEXT NOT NULL REFERENCES servers (id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  compose_project TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  container_count INT NOT NULL DEFAULT 0,
  restart_count INT NOT NULL DEFAULT 0,
  published_ports JSONB NOT NULL DEFAULT '[]'::jsonb,
  last_log_timestamp TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX service_groups_server_compose_idx
  ON service_groups (server_id, compose_project);

CREATE TABLE containers (
  id TEXT PRIMARY KEY,
  service_group_id TEXT NOT NULL REFERENCES service_groups (id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  image TEXT NOT NULL,
  status TEXT NOT NULL,
  health TEXT NOT NULL DEFAULT '',
  cpu_usage_pct DOUBLE PRECISION NOT NULL DEFAULT 0,
  memory_mb DOUBLE PRECISION NOT NULL DEFAULT 0,
  network_mb DOUBLE PRECISION NOT NULL DEFAULT 0,
  restart_count INT NOT NULL DEFAULT 0,
  uptime TEXT NOT NULL DEFAULT '',
  ports JSONB NOT NULL DEFAULT '[]'::jsonb,
  command TEXT NOT NULL DEFAULT '',
  last_seen_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX containers_service_group_name_idx
  ON containers (service_group_id, name);

CREATE TABLE metric_points (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  server_id TEXT NOT NULL REFERENCES servers (id) ON DELETE CASCADE,
  service_group_id TEXT REFERENCES service_groups (id) ON DELETE CASCADE,
  container_id TEXT REFERENCES containers (id) ON DELETE CASCADE,
  metric_key TEXT NOT NULL,
  unit TEXT NOT NULL,
  recorded_at TIMESTAMPTZ NOT NULL,
  value DOUBLE PRECISION NOT NULL
);

CREATE INDEX metric_points_server_key_recorded_at_idx
  ON metric_points (server_id, metric_key, recorded_at DESC);

CREATE INDEX metric_points_container_key_recorded_at_idx
  ON metric_points (container_id, metric_key, recorded_at DESC);

CREATE TABLE log_lines (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  server_id TEXT NOT NULL REFERENCES servers (id) ON DELETE CASCADE,
  service_group_id TEXT REFERENCES service_groups (id) ON DELETE CASCADE,
  container_id TEXT REFERENCES containers (id) ON DELETE CASCADE,
  level TEXT NOT NULL,
  message TEXT NOT NULL,
  container_name TEXT NOT NULL DEFAULT '',
  service_tag TEXT NOT NULL DEFAULT '',
  occurred_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX log_lines_lookup_idx
  ON log_lines (server_id, service_group_id, occurred_at DESC);

CREATE INDEX log_lines_container_lookup_idx
  ON log_lines (container_id, occurred_at DESC);

CREATE TABLE agents (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  server_id TEXT REFERENCES servers (id) ON DELETE SET NULL,
  name TEXT NOT NULL,
  api_key_hash TEXT NOT NULL,
  version TEXT NOT NULL,
  server_name TEXT NOT NULL,
  hostname TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  last_seen_at TIMESTAMPTZ,
  enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE system_onboardings (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  server_id TEXT NOT NULL UNIQUE,
  agent_id TEXT NOT NULL UNIQUE REFERENCES agents (id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK (status IN ('awaiting_connection', 'connected')),
  created_by_user_id TEXT REFERENCES users (id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  connected_at TIMESTAMPTZ
);

CREATE INDEX system_onboardings_tenant_status_idx
  ON system_onboardings (tenant_id, status, created_at DESC);

CREATE TABLE event_logs (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  server_id TEXT NOT NULL REFERENCES servers (id) ON DELETE CASCADE,
  service_group_id TEXT NOT NULL REFERENCES service_groups (id) ON DELETE CASCADE,
  container_id TEXT REFERENCES containers (id) ON DELETE SET NULL,
  event_type TEXT NOT NULL,
  message TEXT NOT NULL,
  entity_name TEXT NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_event_logs_service_time
  ON event_logs (service_group_id, occurred_at DESC);

CREATE INDEX idx_event_logs_container_time
  ON event_logs (container_id, occurred_at DESC);
