-- +migrate Up
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
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, email)
);

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
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE containers (
  id TEXT PRIMARY KEY,
  service_group_id TEXT NOT NULL REFERENCES service_groups (id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  image TEXT NOT NULL,
  status TEXT NOT NULL,
  health TEXT NOT NULL DEFAULT '',
  cpu_usage_pct DOUBLE PRECISION NOT NULL DEFAULT 0,
  memory_mb DOUBLE PRECISION NOT NULL DEFAULT 0,
  restart_count INT NOT NULL DEFAULT 0,
  uptime TEXT NOT NULL DEFAULT '',
  ports JSONB NOT NULL DEFAULT '[]'::jsonb,
  command TEXT NOT NULL DEFAULT '',
  last_seen_at TIMESTAMPTZ
);

CREATE TABLE metric_points (
  id BIGSERIAL PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  server_id TEXT NOT NULL REFERENCES servers (id) ON DELETE CASCADE,
  service_group_id TEXT REFERENCES service_groups (id) ON DELETE CASCADE,
  metric_key TEXT NOT NULL,
  unit TEXT NOT NULL,
  recorded_at TIMESTAMPTZ NOT NULL,
  value DOUBLE PRECISION NOT NULL
);

CREATE INDEX metric_points_server_key_recorded_at_idx
  ON metric_points (server_id, metric_key, recorded_at DESC);

CREATE TABLE log_lines (
  id BIGSERIAL PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  server_id TEXT NOT NULL REFERENCES servers (id) ON DELETE CASCADE,
  service_group_id TEXT REFERENCES service_groups (id) ON DELETE CASCADE,
  container_id TEXT REFERENCES containers (id) ON DELETE CASCADE,
  level TEXT NOT NULL,
  message TEXT NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX log_lines_lookup_idx
  ON log_lines (server_id, service_group_id, occurred_at DESC);

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

-- +migrate Down
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS log_lines;
DROP TABLE IF EXISTS metric_points;
DROP TABLE IF EXISTS containers;
DROP TABLE IF EXISTS service_groups;
DROP TABLE IF EXISTS servers;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;

