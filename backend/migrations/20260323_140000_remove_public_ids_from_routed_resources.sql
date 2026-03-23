-- +migrate Up
DROP INDEX IF EXISTS system_onboardings_server_public_id_key;
DROP INDEX IF EXISTS system_onboardings_public_id_key;
DROP INDEX IF EXISTS containers_public_id_key;
DROP INDEX IF EXISTS service_groups_public_id_key;
DROP INDEX IF EXISTS servers_public_id_key;

ALTER TABLE system_onboardings
  DROP COLUMN IF EXISTS server_public_id,
  DROP COLUMN IF EXISTS public_id;

ALTER TABLE containers
  DROP COLUMN IF EXISTS public_id;

ALTER TABLE service_groups
  DROP COLUMN IF EXISTS public_id;

ALTER TABLE servers
  DROP COLUMN IF EXISTS public_id;

-- +migrate Down
ALTER TABLE servers
  ADD COLUMN IF NOT EXISTS public_id TEXT;

ALTER TABLE service_groups
  ADD COLUMN IF NOT EXISTS public_id TEXT;

ALTER TABLE containers
  ADD COLUMN IF NOT EXISTS public_id TEXT;

ALTER TABLE system_onboardings
  ADD COLUMN IF NOT EXISTS public_id TEXT,
  ADD COLUMN IF NOT EXISTS server_public_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS servers_public_id_key
  ON servers (public_id)
  WHERE public_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS service_groups_public_id_key
  ON service_groups (public_id)
  WHERE public_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS containers_public_id_key
  ON containers (public_id)
  WHERE public_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS system_onboardings_public_id_key
  ON system_onboardings (public_id)
  WHERE public_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS system_onboardings_server_public_id_key
  ON system_onboardings (server_public_id)
  WHERE server_public_id IS NOT NULL;
