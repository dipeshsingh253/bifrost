# 📋 TODO

## 🔥 Current Task
<!-- ONLY ONE task here at a time -->

- [ ] Add targeted regression coverage for empty and replaced Docker snapshots
  - Goal: Lock in the ingest behavior for project replacement, stale service removal, and real compose project discovery.
  - Context: Live debugging fixed real Docker project discovery and stale seeded service cleanup, but there is still no regression coverage around snapshot replacement or around the Docker project filter behavior.
  - Constraints: Keep the tests focused on the current in-memory ingest path and Docker collector rules. Do not broaden scope into frontend error handling or the Postgres migration.
  - Done When: Backend and agent tests cover filtered project discovery, stale service/container removal, and server bundle responses after ingest replaces seed/demo state.
  - Files/Areas: `backend/internal/store`, `backend/internal/httpapi`, `agent/internal/collector`
  - Notes: Promote this immediately after the live-discovery fix so the behavior stays locked before larger persistence work starts.

---

## 📌 Next Tasks (Queue)

- [ ] Surface backend fetch failures in the frontend instead of silently falling back to empty states
  - Goal: Make SSR/API failures visible during development so empty tables are distinguishable from truly empty data.
  - Done When: The frontend logs or renders an explicit development-friendly error state when server-side API fetches fail, instead of always showing generic empty lists.

---

## 🧠 Backlog

- [ ] Move backend runtime storage from the in-memory seed store to PostgreSQL
- [ ] Refactor backend toward the documented handler → service → database structure
- [ ] Replace container-name-keyed metric series with container-ID-keyed contracts
- [ ] Add a real event model instead of deriving all project/container events on read

---

## 🚧 Blocked

- [ ] Full Postgres-backed implementation of the monitoring data plane
  - Blocked Because: The immediate goal is to get the frontend off mocks with the smallest viable backend and agent changes first.
  - Needs: The in-memory contract and ingest path to be working end-to-end before persistence is swapped underneath it.

---

## ✅ Completed (Do not delete)

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
