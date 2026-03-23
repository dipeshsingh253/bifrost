# 📋 TODO

## 🔥 Current Task
<!-- ONLY ONE task here at a time -->

- [ ] Await the next scoped task
  - Goal: Keep the tracker accurate now that the current unblocked queue items have been completed.
  - Constraints: Do not invent a new product task without direction.
  - Done When: A new concrete scoped task is approved and moved into this slot.

---

## 📌 Next Tasks (Queue)

- No queued tasks right now.

---

## 🧠 Backlog

- No unblocked backlog items right now.

---

## 🚧 Blocked

- [ ] Advanced auth features beyond MVP
  - Blocked Because: The approved scope is intentionally narrow: bootstrap admin, login/logout, invite-only viewers, simple `admin`/`viewer` RBAC, and one self-hosted tenant per install.
  - Needs: The base Postgres-backed auth and admin panel flow to be stable first.

- [ ] Fine-grained server-level and service-level permission controls
  - Blocked Because: MVP authorization is tenant-scoped with only two roles and no granular permission matrix yet.
  - Needs: Stable tenant memberships and admin/viewer enforcement before introducing narrower scopes.

---

## ✅ Completed (Do not delete)

- [x] Simplify agent installation to quick copy-paste commands instead of manual config-file setup
  - Notes: Added env-based agent configuration loading so the agent can boot and self-enroll without a handwritten `config.yaml`, added a public backend installer script at `/api/v1/install/agent.sh` that writes an env file, extracts the agent binary from the Docker image, installs a systemd service, and starts it, extended onboarding responses with `install_script_url`, `docker_run_command`, and `systemd_install_command`, reshaped the Add System drawer so Quick Install is the primary step and the raw config/token are pushed into an advanced fallback step, and validated with `go test ./...` in `backend/` and `agent/`, plus `npm test`, `npm run lint`, and `npm run build` in `frontend/`, followed by a live frontend-backed onboarding request that returned the new quick-install commands.

- [x] Reset the local development database to the current schema and validate Add System plus UUID-only monitoring end to end
  - Notes: Stopped the live backend, rebuilt the disposable local `bifrost` database from the checked-in migration `Up` chain so `system_onboardings` and `event_logs` exist and `public_id` columns do not, restarted the backend on `:8080`, bootstrapped a fresh admin through the real frontend proxy on `:3000`, confirmed `GET /api/admin/systems` returns success instead of the prior onboarding-load failure, created a new system through the real frontend Add System proxy, ran a real agent from the generated UUID-only config until it self-enrolled and pushed a snapshot, confirmed the onboarding moved to `connected`, confirmed the server page rendered at `/servers/a0ad9ea6-5d4d-41bf-babc-a540ca2575be`, confirmed `/api/v1/servers` now returns UUID-only server/service/container IDs, spot-checked the live Postgres row IDs directly inside `bifrost-postgres`, and revalidated with `go test ./...` in `backend/`.

- [x] Remove the remaining prefixed/non-UUID ID generators from backend auth, invite, onboarding, and session creation paths
  - Notes: Removed the last prefix-oriented `newResourceID(...)` helper usage from the live backend write paths in Postgres and memory stores, switched bootstrap admin, invite creation, viewer acceptance, onboarding creation, and session creation to direct UUID generation, replaced non-legacy test placeholders like `onb-test` and `srv-missing` with UUID strings, and revalidated with `go test ./...` in `backend/`.

- [x] Replace runtime-generated monitoring resource IDs with canonical UUID strings
  - Notes: Stopped trusting agent-supplied runtime IDs as primary monitoring resource IDs during ingest; the backend now reuses canonical UUIDs for existing services/containers based on stable Docker identity and mints fresh UUIDs when runtime IDs are legacy `svc-*` values or raw Docker container hashes; remapped container metrics and logs onto those canonical IDs before persistence in both memory and Postgres stores; and added backend regressions in `backend/internal/httpapi/router_test.go` and `backend/internal/store/postgres_test.go` to prove legacy runtime IDs are rewritten to UUID-only API/persistence IDs.

- [x] Add targeted regression coverage for empty and replaced Docker snapshots
  - Notes: Tightened the existing backend router regression so the replacement Docker snapshot now uses canonical UUID-style replacement project and container IDs instead of legacy-shaped placeholders; verified the replacement project detail is reachable before the empty snapshot; verified the exact replacement container metric series is present while stale seeded series are pruned; and validated with `go test ./...` in `backend/`.

- [x] Re-run the full Add System validation after the new regression lands
  - Notes: Re-created a fresh empty `bifrost_add_system_revalidation` database, applied the full migration `Up` chain, started an isolated backend on `127.0.0.1:18082` and frontend on `127.0.0.1:3003`, bootstrapped a new admin through the real frontend auth proxy, created a system through the real frontend `/api/admin/systems` proxy, confirmed the onboarding detail reported UUID `id`, `server_id`, and `agent_id` values in `awaiting_connection`, started a real agent from the generated `config.yaml`, confirmed the agent rewrote that config with the rotated long-lived `api_key`, confirmed the onboarding detail flipped to `connected`, and confirmed the target `/servers/<uuid>` page rendered successfully for the connected system. That matches the new automated Postgres regression for the empty-database onboarding case.

- [x] Add a backend regression for Postgres self-enrollment before first server ingest
  - Notes: Added a focused Postgres store integration test in `backend/internal/store/postgres_test.go` that creates a disposable test database from the checked-in migration `Up` chain, bootstraps an admin, creates a pending system onboarding, proves the matching `servers` row does not exist yet, self-enrolls the pending agent successfully, verifies the bootstrap key is invalidated and the rotated key works, and confirms the persisted `agents.server_id` remains `NULL` until first ingest so the old foreign-key failure cannot recur; validated with `go test ./...` in `backend/`.

- [x] Remove the stale frontend `next dev` lock issue if it reproduces during normal local startup
  - Notes: Investigated the reported `.next/dev/lock` failure without changing product code; confirmed an active repo-local `next dev` process was already running from `/home/kakarot/projects/bifrost/frontend`; confirmed `.next/dev/lock` existed alongside that live process; re-ran `cd frontend && npm run dev` and reproduced the lock error only in the presence of that already-running instance; and documented the result as an external/process-state issue rather than a stale lock generated by Bifrost code. No repo changes were required beyond tracker updates.

- [x] Manually validate the Add System flow against a live UUID-only backend
  - Notes: Created a fresh empty `bifrost_add_system_validation` database inside the local `bifrost-postgres` container, applied the full migration `Up` chain, started an isolated backend on `127.0.0.1:18081` and frontend on `127.0.0.1:3002`, bootstrapped a new admin through the real frontend auth proxy, created a system through the real frontend `/api/admin/systems` proxy, confirmed the onboarding detail/list endpoints reported UUID `id`, `server_id`, and `agent_id` values in `awaiting_connection`, found and fixed a live Postgres self-enrollment bug where `agents.server_id` was being set before the corresponding `servers` row existed, reran `go test ./...` in `backend/`, started a real agent from the generated `config.yaml`, confirmed the agent rewrote that config with the rotated long-lived `api_key`, confirmed the onboarding detail flipped to `connected`, and confirmed the target `/servers/<uuid>` page rendered successfully for the connected system through the frontend. Direct headful browser observation of the drawer’s automatic redirect was not performed, but the live polling target and destination route were both verified.

- [x] Rebuild demo/dev seed data and validate the stack from a fresh database
  - Notes: Created a disposable `bifrost_uuid_validation` database inside the local `bifrost-postgres` container, applied the full migration `Up` chain from the checked-in SQL files, started an isolated backend on `127.0.0.1:18080` with `BIFROST_SEED_DEMO_DATA=true`, confirmed the recreated `servers`, `service_groups`, and `containers` rows all used UUID-string `id` values, reran `go test ./...` in both `backend/` and `agent/` plus `npm test`, `npm run lint`, and `npm run build` in `frontend/`, started a production frontend on `127.0.0.1:3001` against that backend, logged in through the real Next auth proxy with the seeded owner account, confirmed the dashboard emitted `/servers/<uuid>` links, and confirmed seeded server/project/container detail pages rendered successfully from the fresh UUID-only database.

- [x] Repair frontend monitoring pages and Add System flow against the UUID-only contract
  - Notes: Audited the monitoring pages, route builders, SSR loaders, and admin API proxies and confirmed they already use canonical route IDs; removed the last stale `server_public_id` dependency from the shared Add System flow helper so connected onboardings now redirect using the canonical UUID `server_id`; updated the onboarding flow tests to use UUID-only payloads and verify redirect behavior stays null until a connected UUID server ID exists; and validated with `npm test`, `npm run lint`, and `npm run build` in `frontend/`.

- [x] Refactor agent enrollment and generated config to use UUID-only identifiers
  - Notes: Added canonical `agent_id` to the generated onboarding YAML and removed the last local `server_id + "-agent"` derivation from the agent runtime; updated the agent config model and checked-in example configs to carry both UUID `agent_id` and UUID `server_id`; made agent startup validate the required identity/config fields before running; switched self-enrollment to submit the configured UUID `agent_id` and `server_id`; switched snapshot ingest payloads to use the configured `agent_id` instead of deriving one from the server; tightened the enroll client to verify the backend response returns the same UUID `agent_id` and `server_id` it requested; added focused agent tests for enrollment persistence and snapshot identity plus backend tests for generated YAML contents; and validated with `go test ./...` in both `agent/` and `backend/`.

- [x] Remove the routed-resource `public_id` model and collapse backend identity onto canonical `id`
  - Notes: Removed `public_id` and `server_public_id` from the backend domain model, monitoring/onboarding response mappers, memory-store lookups, Postgres lookups, and onboarding service views; switched all backend route resolution and JSON responses to canonical `id` values only; removed the Postgres startup backfill path that existed solely for `public_id`; updated seeded backend data and route/onboarding tests to hit canonical IDs directly; added a new migration that drops the routed-resource public-ID columns and indexes; validated with `go test ./...` in `backend/`; and applied the full migration `Up` chain to a disposable Docker Postgres database, confirming every public `id` column remains `text` and zero `public_id` / `server_public_id` columns remain in the final schema.

- [x] Replace every table `id` with a plain UUID string and reset local/dev data
  - Notes: Added a fresh-data reset migration that recreates the full local/dev schema with `TEXT` UUID-string `id` columns for `tenants`, `users`, `tenant_memberships`, `user_sessions`, `user_invites`, `servers`, `service_groups`, `containers`, `metric_points`, `log_lines`, `agents`, `system_onboardings`, and `event_logs`; switched backend seed/runtime ID generation off prefixed IDs and onto random UUID strings, including removal of `server_id + "-agent"` derivation for onboarding-created agents; updated Postgres write paths and read scans for `metric_points`, `log_lines`, and `event_logs` to use explicit UUID-string IDs; updated demo seed data and backend tests to use UUID canonical IDs; validated with `go test ./...` in `backend/`; and applied the full migration `Up` chain to a disposable Docker Postgres database, confirming every public `id` column resolves to `text` in the final schema.

- [x] Audit the full schema and code paths for a UUID-only ID reset
  - Notes: Locked the UUID-only reset scope across every table with an `id` column and every related FK path; confirmed the schema currently spans `tenants`, `users`, `tenant_memberships`, `user_sessions`, `user_invites`, `servers`, `service_groups`, `containers`, `metric_points`, `log_lines`, `agents`, `system_onboardings`, and `event_logs`; confirmed the codebase still relies on prefixed ID generators like `tenant`, `usr`, `tm`, `sess`, `inv`, `srv`, `onb`, and `server_id + "-agent"`; confirmed routed resources still use a mixed `id`/`public_id` model with `server_public_id` in onboarding; confirmed the agent/bootstrap/frontend Add System flows still depend on that split; and translated the rollout into a fresh-data-reset implementation sequence with no legacy-ID compatibility work.

- [x] Define the UUID-only reset direction for IDs and routes
  - Notes: Locked the direction to plain UUID strings in the actual `id` column for every table, no prefixes/suffixes, no routed-resource `public_id` split, no legacy-ID compatibility work for current dev data, and a clean-data reset approach instead of migrating the existing local rows.

- [x] Add regression coverage for route resolution and stale-link handling after the UUID migration
  - Notes: Added a focused backend router regression that proves monitoring routes resolve by UUID public IDs while the old readable server, project, and container IDs now fail with the correct `SERVER_NOT_FOUND`, `PROJECT_NOT_FOUND`, and `CONTAINER_NOT_FOUND` codes; extended the frontend not-found tests so stale legacy project/container route errors stay mapped to the correct page-specific empty states; reran `go test ./...` in `backend/` plus `npm test`, `npm run lint`, and `npm run build` in `frontend/`.

- [x] Update frontend route construction and page loaders to consume UUID public IDs
  - Notes: Added shared frontend monitoring route builders for browser paths and server-side API paths, switched dashboard/server/project/container links over to those helpers, renamed SSR loader route parameters to make the UUID route contract explicit, updated frontend API helpers to build monitoring requests from route-facing IDs instead of interpolating older readable identifiers directly, switched the Add System connection redirect to use `server_public_id` while keeping `server_id` for install-time agent config, added focused frontend tests for the onboarding redirect and monitoring route builders, and validated with `npm test`, `npm run lint`, and `npm run build` in `frontend`.

- [x] Resolve backend lookups and API contracts by UUID public IDs
  - Notes: Switched backend monitoring and onboarding route resolution over to public UUIDs while keeping internal readable IDs behind the repository and ingest layers, updated system onboarding detail/cancel/reissue lookups to use onboarding public IDs, added explicit HTTP response mappers so monitoring `id`, `server_id`, `service_id`, and `container_id` fields now emit public UUID route identifiers, remapped container metric bundle keys and log line resource IDs onto public UUIDs, pinned stable UUID public IDs into seeded demo resources for deterministic backend behavior, updated router and onboarding-service tests to exercise the UUID route contract, and validated with `go test ./...` in `backend/`.

- [x] Introduce UUID-based public IDs for routed resources in the backend data model
  - Notes: Added stable UUID `public_id` fields for servers, services/projects, containers, and system onboardings in the backend domain model; added a new migration for those columns and indexes; generated public IDs for new memory-store, seed, onboarding, and ingest records while preserving existing internal primary keys; backfilled missing Postgres public IDs on store startup so existing rows gain stable opaque identifiers without changing internal relationships; exposed the new UUIDs in server-bundle and onboarding JSON responses; added focused backend tests for onboarding and server bundle public IDs; validated with `go test ./...` in `backend/`; and applied all migration `Up` sections, including the new public-ID migration, to a throwaway Postgres database in Docker.

- [x] Fix the frontend error-state split between missing server and missing project
  - Notes: Added a narrow frontend-only not-found helper to preserve backend `SERVER_NOT_FOUND`, `PROJECT_NOT_FOUND`, and `CONTAINER_NOT_FOUND` distinctions on the project and container detail pages, updated both SSR loaders to return an explicit not-found kind instead of collapsing everything into `server: null`, updated the rendered empty states so project/container misses no longer show the fake `Server not found.` message, added focused frontend tests for the new helper, and validated with `npm test`, `npm run lint`, and `npm run build` in `frontend`.

- [x] Investigate the broken project route and define a UUID-first public ID rollout
  - Notes: Confirmed from the checked-in code that the misleading UI comes from the frontend project/detail SSR pages collapsing `SERVER_NOT_FOUND` and `PROJECT_NOT_FOUND` into the same `server: null` state, which then renders `Server not found.` first; confirmed `srv-dev-server` is a valid seeded demo server ID but `svc-zhiro` does not appear anywhere in the frontend, backend, or seed/demo data in this repository; confirmed project links are built from backend-emitted service IDs rather than hardcoded frontend strings; mapped the next implementation steps into a UUID-first rollout that keeps tenant auth as the real security boundary while replacing guessable routed IDs with opaque public UUIDs.

- [x] Add agent self-enrollment only if the manual credential flow proves too cumbersome
  - Notes: Reintroduced a secure public `POST /api/v1/agents/enroll` bootstrap path, treated pending onboarding secrets as one-time enrollment tokens instead of immediate ingest credentials, blocked snapshot ingest while an agent is still in the bootstrap `pending` state, rotated the token to a real opaque API key on successful self-enrollment in both the memory and Postgres stores without adding new schema, updated generated onboarding YAML and drawer copy to emit `enrollment_token`, taught the agent to exchange that bootstrap token on first start and persist the rotated key back into `config.yaml`, made the Docker install mount writable so the agent can rewrite its config after enrollment, added backend and agent tests for the exchange flow, and validated with `go test ./...` in `backend` and `agent` plus `npm test`, `npm run lint`, and `npm run build` in `frontend`.

- [x] Add cancel/reissue controls for pending system onboarding records
  - Notes: Added pending-only backend cancel and credential reissue actions to the system onboarding service, repository, memory store, and Postgres store; exposed admin-only `POST /api/v1/admin/systems/:systemID/cancel` and `/reissue` routes; kept reissue aligned with the current agent contract by rotating only the opaque API key while preserving the same onboarding, server, and agent identity; added router and service tests covering pending-only access plus old-key invalidation; wired the frontend admin proxies and Add System drawer with Resume/Reissue/Cancel controls for pending records and the waiting state; and validated with `go test ./...` in `backend` plus `npm test`, `npm run lint`, and `npm run build` in `frontend`.

- [x] Add a real event model instead of deriving all project/container events on read
  - Notes: Added a new persisted `event_logs` migration, introduced shared container state-transition event generation for start/stop/restart/health changes, seeded initial events for demo data, emitted new events during both memory-store and Postgres snapshot ingest, switched project/container event reads over to stored event records instead of on-read derivation, added a router regression test proving container events survive a later stop/restart transition, validated with `go test ./...` in `backend`, and applied all migration `Up` sections including the new event table to a throwaway Postgres database in Docker.

- [x] Replace container-name-keyed metric series with container-ID-keyed contracts
  - Notes: Switched backend container metric bundles and per-container history lookups to stable container IDs in both the memory and Postgres stores, updated seeded metric data and router assertions to match the new contract, added a small frontend helper for mapping metric keys back to display labels, updated the server and project stacked charts to render label-aware series while reading ID-keyed payloads, added focused frontend tests for that mapping logic, and validated with `go test ./...` in `backend` plus `npm test`, `npm run lint`, and `npm run build` in `frontend`.

- [x] Refactor backend toward the documented handler → service → database structure
  - Notes: Extracted the admin system onboarding flow into a focused backend service package, moved input trimming plus validation and onboarding response/config shaping out of the HTTP router, wired the existing `/api/v1/admin/systems` create/list/detail handlers through that service so this slice now follows handler → service → database more clearly, added focused service tests for validation and round-trip view shaping, and validated with `go test ./...` in `backend/`.

- [x] Add targeted regression coverage for empty and replaced Docker snapshots
  - Notes: Added a backend router test that ingests a replacement Docker snapshot and then an empty Docker snapshot against the seeded memory store, verified stale project routes disappear, verified server detail and project/standalone-container lists reflect only the current snapshot state, and confirmed stale container metric series are pruned from the bundled server response, then validated with `go test ./...` in `backend/`.

- [x] Surface backend fetch failures in the frontend instead of silently falling back to empty states
  - Notes: Added exported API error classifiers for SSR page loaders, introduced a shared monitoring-unavailable state component, updated the dashboard plus server/project/container pages to preserve true not-found handling while surfacing backend fetch failures as explicit unavailable messages, kept the existing auth redirect behavior by rethrowing unauthorized SSR fetch errors, and validated the frontend with `npm test`, `npm run lint`, and `npm run build`.

- [x] Add targeted regression coverage for the waiting state and resume path
  - Notes: Extracted the Add System drawer’s waiting/resume transition rules into a small shared helper, added a built-in Node test suite covering pending-system resume filtering, resume-step selection, wait-state preservation, and connected-system redirect path generation, wired the drawer to use those tested helpers, added a frontend `npm test` script with Node’s type-stripping runner, and validated the frontend with `npm test`, `npm run lint`, and `npm run build`.

- [x] Validate the full manual onboarding flow end to end
  - Notes: Re-ran `go test ./...` in both `backend/` and `agent/`, re-ran `npm run lint` plus `npm run build` in `frontend/`, created a fresh isolated `bifrost_e2e` Postgres database from the checked-in migration `Up` sections, started an isolated backend on `127.0.0.1:18080` and frontend on `127.0.0.1:3000`, bootstrapped a new admin through the frontend auth proxy, created a system through `/api/admin/systems`, wrote the generated config into a temp agent run directory, ran a real agent binary against that config, confirmed the onboarding status flipped from `awaiting_connection` to `connected`, and verified both `/` and `/servers/<server_id>` rendered the new `Validation VPS` system through the frontend.

- [x] Implement the waiting state and redirect after first connection
  - Notes: Extended the frontend admin system proxy to support both list and detail reads, finished the Add System drawer waiting step with polling plus manual refresh, preserved in-progress wait state across close/reopen, surfaced pending system resume cards, redirected automatically to the connected server detail page on first successful connection, and validated the frontend with `npm run lint` plus `npm run build`.

- [x] Implement the Add System step flow for details, generated config, and install instructions
  - Notes: Replaced the onboarding shell placeholder with a real three-step drawer flow that captures system details, creates a system through the new same-origin `POST /api/admin/systems` proxy, renders copyable generated config and API key data, shows Docker and systemd install tabs with copyable command blocks, and hands the user into a lightweight waiting placeholder, then validated the frontend with `npm run lint` plus `npm run build`.

- [x] Gate the Add System action to admins and add the frontend onboarding shell
  - Notes: Hid the header `Add System` action from viewers, wired it to a new right-side onboarding shell in the shared layout for admins only, kept the shell intentionally narrow to the approved entry experience without implementing the step logic yet, and validated the frontend with `npm run lint` plus `npm run build`.

- [x] Add backend read APIs for pending and recently connected systems
  - Notes: Added tenant-scoped system onboarding list support in both the repository and store layers, exposed admin-only `GET /api/v1/admin/systems` alongside the existing single-item polling endpoint, kept the payload narrow to onboarding metadata and status fields only, validated listing behavior with a router test that covers pending-plus-connected records and viewer denial, and reran `go test ./...` in `backend/`.

- [x] Update snapshot ingest to attach first successful agent activity to the pending system record
  - Notes: Updated both memory and Postgres ingest paths to mark matching onboarding records `connected` on first successful snapshot while preserving the initial `connected_at` timestamp across repeated ingests, added an admin-only `GET /api/v1/admin/systems/:systemID` endpoint so the frontend can poll a single onboarding record, validated the transition with a router test that covers first-ingest and repeat-ingest behavior, and reran `go test ./...` in `backend/`.

- [x] Design and implement the secure backend foundation for system onboarding
  - Notes: Added a `system_onboardings` migration and domain model for pending system installs, introduced an admin-only `POST /api/v1/admin/systems` endpoint that mints opaque API keys server-side, removed the weak public `/api/v1/agents/enroll` path from the router, returned install-ready agent YAML plus `backend_url`, added `BIFROST_AGENT_BACKEND_URL` config support, validated the new admin contract with router tests, reran `go test ./...` in `backend/`, and applied all migration `Up` sections to a throwaway Postgres database in Docker to confirm the new schema works.

- [x] Create a persistent cross-session task tracker
  - Notes: Initial lightweight tracker created before the stricter task workflow was introduced.

- [x] Review the current Bifrost frontend and identify weak component styling
  - Notes: Completed during the frontend polish pass.

- [x] Clone Beszel locally for frontend inspiration and inspect its visual/component patterns
  - Notes: Used as design reference only.

- [x] Refine the Bifrost dashboard shell to better match Beszel's density, spacing, and surface treatment
  - Notes: Applied to the dashboard shell and surrounding layout components.

- [x] Rework the fleet table and metric cards to use a Beszel-inspired component language without copying code
  - Notes: Completed in the frontend redesign pass.

- [x] Tighten the server detail page so charts, stats, and service cards feel consistent with the fleet view
  - Notes: Completed before the current backend integration phase.

- [x] Run frontend lint/build verification after the redesign
  - Notes: Completed during the frontend pass.

- [x] Fix metric cards so charts use the full width and provide readable trend context
  - Notes: Completed in the frontend polish pass.

- [x] Audit frontend, backend, and agent data flow and create a backend/agent integration plan
  - Notes: Discovery completed with subagent analysis. No code changes were made in that step.

- [x] Implement backend server bundle contract for the frontend
  - Notes: Added server bundle response support, widened backend server/log contracts, seeded container metric series, and validated the handler with `go test ./...`.

- [x] Add backend project, container, log, event, and env detail endpoints
  - Notes: Added server-scoped project/container detail routes, project/container metrics and logs, derived event responses, container env data, standalone container seed data, and validated the handlers with `go test ./...`.

- [x] Upgrade agent host collector to send real server metrics and capacity metadata
  - Notes: Replaced placeholder host fields with `/proc`-backed CPU, memory, network, disk I/O, OS, kernel, and capacity data, widened the agent snapshot contract, added host collector parser tests, and validated with `go test ./...`.

- [x] Upgrade agent docker collector to send real container stats and logs
  - Notes: Replaced fake Docker fallbacks with `docker ps -a`, `docker inspect`, `docker stats`, and timestamped `docker logs`, added real container CPU/memory/network/restart/health data plus deduplicated logs, widened the shared container snapshot contract, and validated with `go test ./...` for both agent and backend.

- [x] Wire the frontend pages off mock data onto the backend APIs
  - Notes: Replaced page-level `dummy-data` usage with server-side backend fetches, added a small frontend API layer, kept the existing UI structure, confirmed no page imports from `frontend/lib/dummy-data.ts`, and validated with `npm run lint` plus `npx next build --webpack` after the default Turbopack build hit a sandbox-specific port-binding failure.

- [x] Fix live Docker project discovery and stale seeded service cleanup
  - Notes: Cleared the demo-only agent project whitelist in `agent/config.yaml`, updated backend ingest to prune missing services/logs for each fresh server snapshot, filtered server bundle container metrics to current live container names, validated with `go test ./...` for backend and agent, and verified the live API replaced `service-a/search-stack` with the real local compose projects `zhiro` and `bifrost`.

- [x] Finalize the Postgres-backed auth, tenancy, and RBAC migration plan
  - Notes: Locked the MVP plan to `admin` and `viewer` roles, invite-only viewers, one self-hosted tenant per install, session-based dashboard auth, separate agent API-key auth, and full backend removal of the in-memory runtime path, then translated that into the current implementation sequence in `todo.md`.

- [x] Extend the Postgres schema for sessions, invites, memberships, and full monitoring persistence
  - Notes: Added a forward-only migration for tenant memberships, user sessions, viewer invites, and the missing monitoring persistence columns/indexes needed to remove the backend memory store; validated the migration by applying the `Up` sections of both migrations to a real Postgres test database, rolling both `Down` sections back successfully, and re-running `go test ./...` in `backend/`.

- [x] Replace the backend in-memory store with Postgres-backed repositories
  - Notes: Added a Postgres repository using `database/sql` with the `pgx` stdlib driver, moved backend startup off the seeded memory store, added demo seeding directly into Postgres, fixed agent enrollment persistence, replaced the Postgres read path with direct SQL instead of rebuilding an in-memory snapshot, added a follow-up migration for `servers.cpu_cores`, and validated with `go test ./...` plus a clean Postgres runtime flow covering migrations, backend boot, login, seeded reads, agent enrollment, heartbeat, snapshot ingest, and readback of a newly ingested server.

- [x] Implement bootstrap admin creation and session-based login/logout
  - Notes: Added bootstrap status and bootstrap-admin endpoints, switched dashboard auth to HttpOnly cookie sessions while keeping bearer fallback for existing protected-route tests, added logout and current-session endpoints, implemented Postgres-backed session revocation and first-admin creation, turned demo auth seeding off by default, and validated with `go test ./...` plus a clean Postgres flow covering bootstrap-required status, admin bootstrap, session lookup, logout, login, and session reuse.

- [x] Implement tenant-scoped RBAC for `admin` and `viewer`
  - Notes: Added explicit admin-role checks in the backend router, introduced a minimal admin-only tenant summary endpoint for future admin UI work, added tenant summary repository support for both memory and Postgres stores, preserved viewer access to monitoring read routes, and validated with `go test ./...` plus a live Postgres flow where an admin received `200` on `/api/v1/admin/summary`, a viewer received `403` there, and the same viewer still received `200` on `/api/v1/servers`.

- [x] Implement invite-only viewer onboarding
  - Notes: Added tenant-scoped viewer invite and viewer-access models, implemented Postgres and memory-store invite management plus invite acceptance, exposed admin invite/access/disable/delete routes and public invite detail/accept routes, issued viewer sessions on invite acceptance, and validated with `go test ./...` plus a live Postgres flow covering admin bootstrap, invite creation, invite detail, viewer acceptance, viewer read access, revoked-invite rejection, viewer disable, and viewer deletion.

- [x] Add the frontend auth shell and protected route flow
  - Notes: Replaced the frontend’s hardcoded bearer token path with cookie-aware SSR data fetching, added Next-side auth proxy routes for bootstrap/login/logout/session/invite actions, added login/setup/invite entry pages, wired all dashboard pages through a shared protected-route guard, surfaced auth-service failures on the entry pages instead of crashing, and validated with `npm run lint`, `npm run build`, plus live `curl` checks showing `/` redirects to `/login` when unauthenticated.

- [x] Build the initial admin panel for tenant user management
  - Notes: Added an admin-only `/admin` page with tenant summary cards, invite creation, pending invite revocation, and viewer disable/delete actions, wired those actions through same-origin Next API routes for the existing backend admin endpoints, added admin-aware SSR fetch helpers and role-gated navigation, and validated with `npm run lint`, `npm run build`, plus live `curl` checks showing `/admin` redirects unauthenticated users to `/login` and `/api/admin/access` returns `401` without a session.

- [x] Validate Postgres-backed auth, tenant isolation, and monitoring persistence end to end
  - Notes: Applied the local Postgres migrations, bootstrapped an admin tenant, validated admin and viewer session flows, confirmed admin-only `/admin` access and viewer redirect behavior, verified cross-tenant server isolation with direct Postgres fixture data, enrolled and ran a real agent binary against the Postgres-backed backend, confirmed persisted server/service/container/metric/log rows in Postgres plus dashboard/API readback, reran `go test ./...` for backend and agent, reran frontend `npm run lint` and `npm run build`, and fixed a dev-time dynamic `<title>` warning discovered during live page validation.

---

## ⚙️ Rules (STRICT)

- Only ONE task in **Current Task**
- Always move task from Queue → Current before starting
- Always follow:
  - Goal
  - Constraints
  - Done When
- Never mark complete without validation
- Always update this file after work
- Keep tasks small and specific (1–2 hours max)

---

## 🧠 How Codex Should Use This

- Read `PRD.md` first
- Read this file before starting work
- Only work on **Current Task**
- If unclear → ask questions
- If task is large → break it down first
- After completion:
  - verify “Done When”
  - move task to Completed
  - pick next task
