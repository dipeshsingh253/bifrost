1. Copy [agent/config.example.yaml](/home/kakarot/projects/bifrost/agent/config.example.yaml) to `agent/config.yaml` and replace the backend URL, server identity, and API key with your local values.
2. Start Postgres with `docker-compose up -d postgres` if you want the database container ready for the next backend storage pass.
3. Install frontend dependencies in [frontend/package.json](/home/kakarot/projects/bifrost/frontend/package.json) with `npm install`.
4. Run the backend from [backend/main.go](/home/kakarot/projects/bifrost/backend/main.go) using `go run .` inside `backend/`.
5. Run the frontend from [frontend/pages/index.tsx](/home/kakarot/projects/bifrost/frontend/pages/index.tsx) using `npm run dev` inside `frontend/`.
6. Run the agent from [agent/main.go](/home/kakarot/projects/bifrost/agent/main.go) using `go run .` inside `agent/`.
7. Verify the new Bifrost frontend by running `cd frontend && npm run dev` and opening `http://localhost:3000` in a browser.
8. Click on any server row in the dashboard to verify the server detail page loads with charts and service groups.
9. Verify the dark theme matches the Beszel screenshots (dark background, green progress bars, area charts with gradient fills).
10. Navigate to a server details page and verify the new "Services" section renders below the Metrics section with accurate stats (Total Projects, Total Containers, Unhealthy, Exposed Ports).
11. Confirm the "Projects" and "Standalone Containers" previews dynamically hide or show based on the presence of services and are limited to 3 items maximum.
12. Click on a Project link in the Services section to verify it navigates to the new Project Detail Page.
13. On the Project Detail Page, verify that container resource usage is grouped accurately and the live simulated logs appear.
14. Click on a Container from the Project Detail Page or the Services standalone list to verify it navigates to the Container Detail Page.
15. On the Container Detail Page, verify container-specific metrics, details, and isolated logs load properly.
16. Review the updated [todo.md](/home/kakarot/projects/bifrost/todo.md) and confirm the current backend contract task before implementation starts.
17. Run the backend from [backend/main.go](/home/kakarot/projects/bifrost/backend/main.go) with `cd backend && go run .`.
18. Log in against `POST /api/v1/auth/login` using `owner@bifrost.local` / `bifrost123` and capture the bearer token from the response.
19. Call `GET /api/v1/servers/srv-dev-server` with the bearer token and verify the response now includes `server`, `services`, `metrics`, and `containerMetrics`.
20. Call `GET /api/v1/services/svc-service-a/logs` with the bearer token and verify each log line includes `containerName` and `serviceTag` for the frontend log viewer.
21. Call `GET /api/v1/servers/srv-dev-server/projects` with the bearer token and verify the response includes the seeded compose projects under `projects`.
22. Call `GET /api/v1/servers/srv-dev-server/projects/svc-service-a/metrics` and verify the returned container metric bundle includes `service-a-*` containers but excludes containers from other projects.
23. Call `GET /api/v1/servers/srv-dev-server/projects/svc-search-stack/events` and verify the response includes derived events such as `restart`, `health_change`, or `crash`.
24. Call `GET /api/v1/servers/srv-dev-server/containers?standalone=true` and verify the seeded standalone container `edge-proxy-1` is returned.
25. Call `GET /api/v1/servers/srv-dev-server/containers/ctr-edge-proxy-1/env` and verify the response includes derived environment fields such as `CONTAINER_NAME`, `PORT`, and `SERVICE_NAME`.
26. Run the agent from [agent/main.go](/home/kakarot/projects/bifrost/agent/main.go) with `cd agent && go run .` after configuring [agent/config.yaml](/home/kakarot/projects/bifrost/agent/config.yaml).
27. After the agent posts at least two snapshots, call `GET /api/v1/servers/<your-server-id>` and verify the returned `server` object now includes non-placeholder `cpu_usage_pct`, `network_rx_mb`, `network_tx_mb`, `kernel`, `cpu_model`, `cpu_cores`, `cpu_threads`, `total_memory_gb`, and `total_disk_gb`.
28. Verify the bundled `metrics` array for that server now includes `cpu_usage_pct`, `network_rx_mb`, `network_tx_mb`, `disk_read_mb`, and `disk_write_mb` points coming from the agent.
29. With Docker running on the monitored host, wait for the agent to post a snapshot and call `GET /api/v1/servers/<your-server-id>` again to verify service containers now include non-zero `cpu_usage_pct`, `memory_mb`, `network_mb`, `restart_count`, normalized `status`, and `health`.
30. Call `GET /api/v1/servers/<your-server-id>/projects/<project-id>/logs` and verify the response now contains real container logs instead of the old fallback line.
31. Call `GET /api/v1/servers/<your-server-id>/containers/<container-id>/metrics` and verify the returned network history is no longer all zeros for active containers.
32. Start the backend and frontend together, open `http://localhost:3000`, and verify the dashboard now loads from the backend without relying on mock data.
33. Open `/servers/<your-server-id>` and verify server charts, service previews, and stacked container charts render from live backend responses.
34. Open `/servers/<your-server-id>/projects/<project-id>` and verify the logs and recent events panels show backend data instead of generated mock content.
35. Open `/servers/<your-server-id>/containers/<container-id>` and verify container logs, chart history, and environment variables are populated from the backend endpoints.
36. Run `docker ps -a --format '{{json .}}'` on the monitored host and note each container's `com.docker.compose.project` label so you can compare it against [agent/config.yaml](/home/kakarot/projects/bifrost/agent/config.yaml).
37. If real Docker projects are missing from Bifrost, check whether `docker.include_projects` in [agent/config.yaml](/home/kakarot/projects/bifrost/agent/config.yaml) still whitelists only demo project names.
38. Call `GET /api/v1/servers/srv-dev-server/projects` after a fresh agent snapshot and verify the response no longer includes stale seeded projects that are absent from the current Docker state.
39. Refresh `http://localhost:3000/servers/srv-dev-server` and verify the Services section now lists the live compose projects from this machine, including `zhiro`, instead of the old demo projects.
40. Open `http://localhost:3000/servers/srv-dev-server/projects` and verify the project list matches `GET /api/v1/servers/srv-dev-server/projects`, with no stale `service-a` or `search-stack` entries remaining.
