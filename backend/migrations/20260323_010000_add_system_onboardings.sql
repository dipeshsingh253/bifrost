-- +migrate Up
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

-- +migrate Down
DROP INDEX IF EXISTS system_onboardings_tenant_status_idx;
DROP TABLE IF EXISTS system_onboardings;
