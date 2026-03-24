# Bifrost

Bifrost is a lightweight server and container monitoring product with three parts:

- `backend/`: Go API and ingest service
- `agent/`: Go collector for host metrics, Docker state, and logs
- `frontend/`: Next.js dashboard

## Current Stack

- Backend: Go + Gin
- Agent: Go
- Frontend: Next.js Pages Router + Tailwind CSS
- Database target: PostgreSQL
- Current runtime storage: in-memory backend store seeded for local development

## Local Development

Requirements:

- Go
- Node.js 22+
- Docker
- GitHub CLI and Git if you want to publish changes

Run locally:

```bash
docker-compose up -d postgres

cd backend
cp .env.example .env
go run .

cd ../frontend
npm install
cp .env.example .env.local
npm run dev

cd ../agent
go run .
```

Default local URLs:

- Frontend: `http://localhost:3000`
- Backend: `http://localhost:8080`

## Environment Files

The frontend and backend are both environment-driven now.

- `backend/.env.example` shows the backend variables. The backend loads `.env` first and `.env.local` second, while still letting real process environment variables win.
- `frontend/.env.example` shows the frontend variables. Next.js loads `frontend/.env.local` natively in local development.
- Production should use the same variable names. The difference should be the values you inject, not separate dev-only code paths.

## Agent Config

See [`agent-installation.md`](agent-installation.md) for:

- Docker installation
- systemd installation
- generated onboarding commands
- `config.yaml` fields and filter behavior

Copy `agent/config.example.yaml` to `agent/config.yaml` if needed.

The default local config is set up to:

- push snapshots to `http://localhost:8080`
- collect host metrics
- collect Docker metrics and logs
- allow all compose projects when `include_projects: []`

## Validation

Backend and agent:

```bash
cd backend && go test ./...
cd agent && go test ./...
```

Frontend:

```bash
cd frontend && npm run lint
cd frontend && npm run build
```

## Notes

- The frontend now reads live backend data instead of page-level mock imports.
- The agent now collects live host and Docker state from the current machine.
- `next-steps.md` is intentionally short-lived and should be overwritten with only the latest manual follow-up steps.
