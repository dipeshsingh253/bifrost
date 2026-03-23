-- +migrate Up
ALTER TABLE servers
  ADD COLUMN cpu_cores INT NOT NULL DEFAULT 0;

-- +migrate Down
ALTER TABLE servers
  DROP COLUMN IF EXISTS cpu_cores;
