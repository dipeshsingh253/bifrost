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
cd backend
cp .env.example .env
# point BIFROST_DATABASE_URL at your PostgreSQL instance
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

## Production Deploy

The central deployment only needs two services:

- `bifrost-backend`
- `bifrost-frontend`

The agent is installed separately on each monitored machine and does not run in the central compose stack.

To deploy with Docker Compose:

```bash
cp .env.example .env
docker network create internal_net # only once if it does not already exist
```

Deployment commands:

```bash
# 1. Pull the latest images
docker compose pull

# 2. Apply database migrations
docker compose run --rm bifrost-backend migrate -path /app/migrations -database "$BIFROST_DATABASE_URL" up

# 3. Start the services
docker compose up -d
```

Notes:

- `docker-compose.yml` assumes PostgreSQL is already running elsewhere and reachable through `BIFROST_DATABASE_URL`.
- The backend image now includes the `migrate` CLI, so you can also run `docker exec -it bifrost-backend migrate -path /app/migrations -database "$BIFROST_DATABASE_URL" up` after the backend container is running.
- `BIFROST_API_BASE_URL` should stay `http://bifrost-backend:8080` inside the compose network so the frontend can reach the backend internally.
- `BIFROST_AGENT_BACKEND_URL` must be the public backend URL agents should call, for example `https://bifrost.example.com`.
- The GitHub Actions workflow at `.github/workflows/build-and-push-images.yml` publishes `ghcr.io/<owner>/bifrost-backend`, `ghcr.io/<owner>/bifrost-frontend`, and `ghcr.io/<owner>/bifrost-agent` on pushes to `main`.

## Notes

- The frontend now reads live backend data instead of page-level mock imports.
- The agent now collects live host and Docker state from the current machine.
- `next-steps.md` is intentionally short-lived and should be overwritten with only the latest manual follow-up steps.
