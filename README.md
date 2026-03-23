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
go run .

cd ../frontend
npm install
npm run dev

cd ../agent
go run .
```

Default local URLs:

- Frontend: `http://localhost:3000`
- Backend: `http://localhost:8080`

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
