# Agent Installation

This document covers:

- Docker installation
- systemd installation
- `config.yaml` structure
- Docker include/exclude behavior

The Bifrost agent can run in 2 supported modes:

- Docker container
- systemd service on the host

## Prerequisites

- A running Bifrost backend
- A created system onboarding from the Bifrost UI
- Docker if you want the Docker install method
- `systemd` if you want the host install method

The Add System flow in the UI generates the exact values needed for:

- `agent_id`
- `server_id`
- `server_name`
- `tenant_id`
- `backend_url`
- `enrollment_token`

## Recommended Flow

Use the `Add System` action in the UI.

It generates:

- a ready-to-run Docker command
- a ready-to-run systemd install command
- a matching `config.yaml`

Prefer using the generated command from the UI instead of manually typing IDs and tokens.

## Docker Install

The Docker flow expects a published agent image:

- `bifrost-agent:latest`

Example:

```bash
docker run -d \
  --name bifrost-agent \
  --restart unless-stopped \
  --network host \
  --pid host \
  --uts host \
  -v /:/hostfs:ro \
  -v bifrost-agent-data:/var/lib/bifrost-agent \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -e BIFROST_CONFIG_PATH='/var/lib/bifrost-agent/config.yaml' \
  -e BIFROST_HOST_ROOT='/hostfs' \
  -e BIFROST_AGENT_ID='...' \
  -e BIFROST_SERVER_ID='...' \
  -e BIFROST_SERVER_NAME='...' \
  -e BIFROST_TENANT_ID='...' \
  -e BIFROST_BACKEND_URL='http://your-backend:8080' \
  -e BIFROST_ENROLLMENT_TOKEN='...' \
  -e BIFROST_COLLECT_HOST='true' \
  -e BIFROST_COLLECT_DOCKER='true' \
  -e BIFROST_COLLECT_LOGS='true' \
  -e BIFROST_DOCKER_INCLUDE_ALL='true' \
  bifrost-agent:latest
```

Notes:

- The container writes its config into `/var/lib/bifrost-agent/config.yaml`
- The first boot self-enrolls using `enrollment_token`
- After enrollment, the agent persists the rotated long-lived `api_key`
- The Docker socket is mounted read-only so the agent can inspect containers and collect logs
- The host root is bind-mounted and the container joins the host PID/UTS namespaces so host metrics reflect the real machine instead of the container

If you previously ran a broken Docker install, clear old state before retrying:

```bash
docker rm -f bifrost-agent 2>/dev/null || true
docker volume rm -f bifrost-agent-data 2>/dev/null || true
```

## systemd Install

The host install flow uses the backend installer script:

```bash
curl -fsSL 'http://your-backend:8080/api/v1/agent/install.sh' | sudo env \
  BIFROST_AGENT_ID='...' \
  BIFROST_SERVER_ID='...' \
  BIFROST_SERVER_NAME='...' \
  BIFROST_TENANT_ID='...' \
  BIFROST_BACKEND_URL='http://your-backend:8080' \
  BIFROST_ENROLLMENT_TOKEN='...' \
  sh
```

What it does:

- creates agent directories
- writes the environment file
- installs the `bifrost-agent` binary
- creates `bifrost-agent.service`
- enables and starts the service

Useful commands:

```bash
sudo systemctl status bifrost-agent --no-pager -l
sudo journalctl -u bifrost-agent -n 100 --no-pager -l
sudo systemctl restart bifrost-agent
```

## config.yaml

The finalized config format is:

```yaml
agent_id: 0a439dad-7646-497d-940c-fbf79cc5a6a3
server_id: 8a3b22fb-3ff7-4e33-bf27-81e693a60982
server_name: localhost
tenant_id: 15d0a6bd-c94e-46cf-830c-ba71af57b524

backend_url: http://localhost:8080
enrollment_token: bootstrap-token
poll_interval_seconds: 10

collectors:
  host: true
  docker: true
  logs: true

docker:
  include_all: true
  include_projects: []
  include_containers: []
  exclude_projects: []
  exclude_containers: []

logs:
  max_lines_per_fetch: 200
```

After first enrollment, the agent replaces `enrollment_token` with `api_key`.

## Field Meanings

Top-level identity:

- `agent_id`: UUID identity for this agent
- `server_id`: UUID identity for the monitored system
- `server_name`: display name shown in the UI
- `tenant_id`: tenant UUID

Connection:

- `backend_url`: Bifrost backend base URL
- `enrollment_token`: one-time bootstrap token from onboarding
- `api_key`: long-lived agent key written after successful enrollment
- `poll_interval_seconds`: snapshot interval

Collectors:

- `collectors.host`: collect host metrics
- `collectors.docker`: collect Docker/container/service state
- `collectors.logs`: collect Docker logs

Docker filtering:

- `docker.include_all`: default include mode
- `docker.include_projects`: compose project names to include
- `docker.include_containers`: standalone or container names to include
- `docker.exclude_projects`: compose project names to exclude
- `docker.exclude_containers`: container names to exclude

Logs:

- `logs.max_lines_per_fetch`: initial tail count for each container before cursor-based fetching takes over

## Docker Filter Rules

Matching is exact string matching only.

Use:

- compose project names in `include_projects` and `exclude_projects`
- container names in `include_containers` and `exclude_containers`

Behavior:

- if `docker.include_all: true`
  - all Docker containers are monitored first
  - then `exclude_projects` and `exclude_containers` are applied
- if `docker.include_all: false`
  - only containers matching `include_projects` or `include_containers` are monitored
  - then `exclude_projects` and `exclude_containers` are applied
- excludes always win
- logs follow the same filtered container set as Docker monitoring

## Defaults

Defaults used by the agent:

```yaml
poll_interval_seconds: 10

collectors:
  host: true
  docker: true
  logs: true

docker:
  include_all: true
  include_projects: []
  include_containers: []
  exclude_projects: []
  exclude_containers: []

logs:
  max_lines_per_fetch: 200
```

## Local Development Image Build

If you need a local Docker image for development:

```bash
docker build -t bifrost-agent:latest agent
```

The image definition is in [`agent/Dockerfile`](agent/Dockerfile).

## Validation

After installation:

1. Check agent logs
2. Wait for the onboarding status to move from `awaiting_connection` to `connected`
3. Open the new server page in the UI

Docker:

```bash
docker logs bifrost-agent
docker ps
```

systemd:

```bash
sudo systemctl status bifrost-agent --no-pager -l
sudo journalctl -u bifrost-agent -n 100 --no-pager -l
```
