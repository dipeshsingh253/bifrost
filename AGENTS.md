# AGENTS.md

## Overview

**Project Name**: Bifrost

**Description**: Bifrost is a server and container monitoring product. The root project contains the backend, agent, and frontend. The goal is to ship a fast production ready product with low complexity, predictable architecture, and clear upgrade paths.

**Type**: Full-stack product

**Stage**: Early development

---

## Stack

**Backend**: Go 1.26+, Gin 1.12+  
**Agent**: Go 1.26+  

**Frontend**: Next.js 16.x (Pages Router)  
**Runtime**: Node.js 22 (LTS)  
**Styling**: Tailwind CSS 4.x  

**Database**: PostgreSQL 18+  
**Logs (MVP)**: PostgreSQL (hot storage)  

**Package Manager**: npm or pnpm  
**Testing**: go test  

**Linting**: ESLint (frontend), gofmt (Go)  
**Formatting**: Prettier (frontend), gofmt (Go)  

**Migrations**: golang-migrate (CLI)  
**DB access**: plain `database/sql` now, `sqlc` later if needed  

**Other**: Docker, Docker Compose

---

## Philosophy

Build the simplest thing that works well.

Prefer:
- fast delivery
- low complexity
- clear and readable code
- small, focused changes
- maintainable architecture
- predictable behavior
- scalability through good structure, not over-engineering
- upgrade paths without premature scaling

Avoid:
- over-engineered solutions
- unnecessary abstractions
- complex patterns without clear benefit
- designing for hypothetical scale

Good structure, clear file organization, and simple abstractions (like basic parent-child patterns) are enough to scale the project effectively.

Do not optimize for enterprise-scale complexity in MVP.

When in doubt:
- choose the simpler solution
- follow existing patterns in the codebase
- ask before making uncertain or impactful decisions

---

## Product Context

Bifrost is a simple server and container monitoring product built for fast setup and clear visibility.

**What Bifrost is**:

- a monitoring and visibility tool for VPS-based workloads
- a way to see servers, containers, services, logs, and basic metrics in one place
- a product designed for simple deployment and operational clarity
- a tool for small teams, indie hackers, and growing startups
- a system optimized for fast production ready execution and gradual scaling

**What Bifrost should prioritize**:

- easy installation
- low operational complexity
- clear backend and agent flow
- simple, readable UI
- predictable pricing tied to servers and retention

**What Bifrost is not**:

- not a full observability platform
- not a highly customizable dashboard builder
- not an enterprise-first system in MVP
- not a complex event pipeline or analytics platform
- not a product that should depend on heavy infrastructure early

**Product direction**:

- self-hosted free tier for adoption and trust
- hosted cloud plans for convenience and retention
- simple server-based pricing
- support for future upgrades without forcing premature architecture

---

## Source of Truth

Use the `/knowledge` directory as the main source of truth for product and architectural decisions.

Before making decisions about architecture, flow, naming, pricing-related logic, or product behavior:

- read the relevant files in `/knowledge`
- follow documented decisions
- do not invent a different architecture unless explicitly asked

If code and `/knowledge` conflict, raise it and ask first.

---

## Structure

Top-level directories:

- `agent/` → Go agent
- `backend/` → Go backend and API
- `frontend/` → Next.js frontend
- `knowledge/` → project knowledge, decisions, tradeoffs, and planning docs

Keep responsibilities separated by directory. Do not mix backend, agent, and frontend logic unnecessarily.

---

## Local Development

```bash
# First time setup
cd backend && go mod download
cd frontend && npm install
docker-compose up -d postgres

# Running locally (3-4 terminals)
docker-compose up postgres           # terminal 1
cd backend && go run .               # terminal 2
cd frontend && npm run dev           # terminal 3
cd agent && go run .                 # terminal 4 (optional)
````

**Access**: Frontend [http://localhost:3000](http://localhost:3000), Backend [http://localhost:8080](http://localhost:8080)

---

## Database Migrations

**Tool**: golang-migrate CLI  
**Location**: `backend/migrations/`  
**Naming**: `YYYYMMDD_HHMMSS_description.sql`

**Structure**:

- each migration file contains both `up` and `down` sections (similar to Alembic style)

**Rules**:

- always include both up and down logic
- never modify existing migrations after commit
- test migrations locally before committing

**Ask first before**:

- adding/removing tables
- changing column types
- adding/removing indexes on large tables
- any schema changes affecting pricing or retention logic

---

## API Response Format

**Success**:

```json
{
  "success": true,
  "data": {}
}
```

**Error**:

```json
{
  "success": false,
  "error": {
    "message": "Human-readable error",
    "code": "ERROR_CODE",
    "details": {}
  }
}
```

**Status codes**: 200, 201, 204, 400, 401, 403, 404, 409, 422, 429, 500

---

## Standards

### Naming

Use clear, boring, predictable names. Prefer descriptive names over clever abbreviations. Avoid generic names like `Manager`, `Helper`, `Util` unless justified.

### Comments

Comments should be intentional and useful.

For orchestration methods:

- explain high-level flow in plain language
- must always include edge cases or non-standard decisions

For helper methods:

- one or two lines only
- focus on edge cases or important decisions

Do not comment obvious code.

### Types

**Go**: explicit types, minimal interfaces, avoid premature abstraction  
**Frontend**: explicit props and data flow, avoid unnecessary layers

---

## Rules

### Do

- keep diffs small and focused
- preserve existing structure unless strong reason not to
- ask first when uncertain
- follow locked stack and architecture
- use `/knowledge` before making decisions
- keep APIs simple and REST-based
- keep agent lightweight and reliable
- keep frontend simple and operationally clear
- suggest improvements freely
- explain assumptions

### Don't

- do not add dependencies without asking
- do not make large refactors without asking
- do not change pricing logic without asking
- do not change auth or tenant model without asking
- do not apply schema changes without asking
- do not delete files without asking
- do not introduce new infrastructure casually
- do not over-engineer for future scale
- do not add heavy abstraction patterns
- do not introduce design system-heavy frontend

---

## Locked Product Decisions

- Go agent
- Go backend with Gin
- PostgreSQL primary database
- Next.js frontend with Pages Router
- PostgreSQL for logs in MVP

---

## Backend Guidance

**Organization**: Feature-based modules (auth, servers, metrics, logs, pricing)  
**Pattern**: handler → service → database

Keep aligned with:

- Gin REST APIs
- multi-tenant design
- PostgreSQL as source of truth

Avoid excessive layering and speculative abstractions.

---

## Agent Guidance

Responsibilities:

- collect metrics
- inspect containers
- group services
- collect logs    
- batch and push to backend

Prefer
- stable behavior
- low memory usage
- predictable retries

Avoid unnecessary complexity.

---

## Frontend Guidance

Use Next.js Pages Router with Tailwind CSS.

Priorities:
- simple and modern UI
- clear navigation
- usability over abstraction

Design rule:
- always ask for design files, screenshots, or clear UI references before implementing or modifying UI
- do not assume or invent UI/UX without confirmation

State:

- React hooks for local state
- Context for auth
- no Redux unless needed

The UI must look like following images but create components tailored to our use case:
![[beszel-dashboard.png]]

![[beszel-server-page.png]]

---

## Testing

Keep testing practical and focused.

Goals:

- avoid basic errors
- ensure code compiles and runs correctly
- catch missing imports, wrong method names, broken paths

Rules:

- use `go test` for backend and agent
- add tests for important logic when useful
- no heavy testing setup unless asked
- frontend has no locked testing framework yet

Validation expectations:

Backend / Agent:

- verify imports and references
- ensure code compiles
- run `go test` where applicable

Frontend:

- ensure build works (`npm run build`)
- ensure lint passes (`npm run lint`)
- ensure dev server runs (`npm run dev`)
- verify pages load without crashes

General:

- never leave broken builds or lint errors
- ensure code is properly wired
- run minimal checks before completing work

Manual testing:

- user flow testing is manual
- always write clear steps after task completion

Next steps file:

- always replace `next-steps.md` with only the latest relevant manual steps
- keep steps simple, ordered, actionable
- remove prior content when the file is updated

---

## Errors

Use explicit, readable error handling. Do not hide failures. Add context when useful.

---

## Permissions

### Allowed

- small changes
- local refactors
- improving naming/comments
- adding validation

### Ask First

- dependencies
- schema changes
- migrations
- deleting files
- pricing/auth changes
- large refactors
- new infrastructure

---

## Commands

### Frontend

```bash
npm install / pnpm install
npm run dev
npm run build
npm run lint
```

### Backend

```bash
go run .
go test ./...
migrate -path backend/migrations -database <DB_URL> up
```

### Agent

```bash
go run .
go test ./...
```

---

## When Stuck

Check `/knowledge`, inspect code, prefer simple solution, ask when unsure.

---

## Notes

Expected working style: moderately flexible but conservative.

Default behavior:

- do not surprise
- do not overbuild
- do not assume too much

Suggest improvements freely, keep implementation practical, ask first when risk exists.

### Next Steps Workflow

Always overwrite `next-steps.md` with the latest manual steps only.

Prefer manual steps for:

- browser validation
- `.env` updates
- third-party setup
- end-to-end testing

Do not try to automate these.

Keep steps simple and actionable.
Keep the file short and current.

---

**Last Updated**: 2026-03-21  
**Maintained By**: Dipesh
