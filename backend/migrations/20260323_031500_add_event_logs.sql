-- +migrate Up
CREATE TABLE event_logs (
  id BIGSERIAL PRIMARY KEY,
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

-- +migrate Down
DROP TABLE IF EXISTS event_logs;
