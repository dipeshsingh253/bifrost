Here is your cleaned and rewritten version without emojis, with improved structure and readability while keeping everything intact:

---

# Final Locked Tech Stack

This is the finalized tech stack based on all decisions and tradeoffs.

Optimized for:

* Fast MVP
* Low complexity
* Smooth scaling later
* Alignment with pricing model

---

# Core Architecture

Agent (per VPS)
→ HTTPS (push)
→ Backend (API + Control Plane)
→ Postgres (primary DB + hot logs)
→ Frontend (Dashboard)

Planned later (not part of MVP):

* Object Storage (R2 / S3) for cold logs
* Redis for background jobs

---

# 1. Agent

Final choice: Go

Runs as:

* systemd service (primary)
* Docker container (optional)

Responsibilities:

* Collect host metrics (CPU, memory, disk, network)
* Inspect Docker (containers, stats)
* Group services via compose labels
* Collect logs (tail-based)
* Batch and push data

Key decisions:

* Push model (agent → backend)
* API key authentication
* Batch interval: ~5–10 seconds

---

# 2. Backend (API + Control Plane)

Final choice: Go (Fiber or Gin)

Responsibilities:

* Authentication and RBAC
* Server and agent management
* Ingest metrics and logs
* Provide query APIs for UI
* Retention cleanup
* Multi-tenant isolation

Key decisions:

* Single codebase for all tiers
* Multi-tenancy via `tenant_id`
* REST API (no gRPC for MVP)

---

# 3. Database

Final choice: PostgreSQL

Used for:

* Users
* Servers
* Services (compose groups)
* Containers
* RBAC
* Metrics (time-series for MVP)
* Logs (hot storage)

Key decisions:

* No TSDB initially
* Proper indexing for performance
* Time-based cleanup jobs

---

# 4. Logs System

## Phase 1 (MVP): Postgres only

Why:

* Simplest approach
* Fastest to build
* Sufficient for 3–7 day retention

Design:
Indexed by:

* server_id
* service_name
* container_id
* timestamp

Search:

* Time range
* Service/container filters
* Keyword search using `ILIKE`

Retention:

* Strict deletion based on plan

---

## Phase 2 (planned)

Hybrid logs:

* Hot storage → Postgres
* Cold storage → R2 / S3 / MinIO

Design:

* Compressed log chunks
* Metadata stored in Postgres
* Fetch and filter on demand

---

# 5. Metrics System

Stored in Postgres

Data:

* Server metrics
* Container metrics

Design:

* Simple time-series table
* Indexed by (entity_id + timestamp)

Retention:

* Enforced via cleanup jobs

---

# 6. Background Jobs

Not included in MVP

Planned addition:

* Redis (or Postgres queue initially)

Use cases:

* Retention cleanup
* Log compaction
* Alerts
* Async processing

---

# 7. Frontend

Final choice: Next.js (React)

Responsibilities:

* Server overview
* Service grouping view
* Container drill-down
* Logs viewer
* Basic metrics charts

Key decisions:

* No complex dashboards
* Prioritize usability over flexibility

---

# 8. Auth and RBAC

MVP:

* Email and password
* JWT/session-based auth

Roles:

* Owner
* Admin
* Member
* Viewer

Later:

* OAuth (Google, GitHub)
* SSO (enterprise)

---

# 9. Communication (Agent → Backend)

Protocol:

* HTTPS (REST)

Format:

* JSON

Authentication:

* API key per agent

Behavior:

* Batch every 5–10 seconds
* Retry on failure

---

# 10. Multi-tenancy

Final approach:

* Single database
* `tenant_id` used everywhere

Supports:

* Multi-tenant cloud
* Single-tenant enterprise deployments

---

# 11. Deployment Strategy

## Self-host (Free)

* Docker Compose
* Includes:

  * Backend
  * Frontend
  * Postgres

## Cloud (Product)

* Start with single VPS or simple infra
* Optional managed Postgres
* No Kubernetes initially

## Enterprise

* Same stack
* Docker Compose or Helm later

---

# 12. Pricing and Tech Alignment

Free (self-host):

* Local Postgres
* Short retention

Starter:

* Limited servers
* Small retention

Pro:

* Longer retention
* More data storage

Enterprise:

* Configurable storage
* Optional object storage

Key insight:
Retention is the primary cost control lever

---

# Final Summary

Stack:

* Agent → Go
* Backend → Go
* Database → Postgres
* Logs → Postgres (MVP), object storage later
* Frontend → Next.js
* Queue → None (Redis later)

Philosophy:

* Build simple
* Optimize later
* Avoid enterprise complexity
* Design for upgrade paths, not premature scale

---

# Outcome

This stack will:

* Enable fast MVP development
* Support real users
* Scale gradually without requiring rewrites
