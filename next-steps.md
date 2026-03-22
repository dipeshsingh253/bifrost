1. Start the backend with `cd backend && go run .`.
2. Start the frontend with `cd frontend && npm run dev`.
3. Start the agent with `cd agent && go run .`.
4. Open `http://localhost:3000/servers/srv-dev-server` and verify the Services section shows the live Docker projects from this machine, including `zhiro` and `bifrost`, with no `service-a` or `search-stack`.
5. Open `http://localhost:3000/servers/srv-dev-server/projects` and verify the project list matches the live backend response.
6. The current task in `todo.md` is regression coverage for Docker snapshot replacement and stale-service cleanup. Work from that task next.
