# Next Steps

1. Copy [.env.example](/home/kakarot/projects/bifrost/.env.example) to `.env` and set the production values for your public backend URL, JWT secret, and shared PostgreSQL connection string.
2. Create the external Docker network once with `docker network create internal_net` if it does not already exist on the server.
3. From [docker-compose.yml](/home/kakarot/projects/bifrost/docker-compose.yml), run `docker compose pull` and `docker compose up -d`.
4. Open the deployed frontend, sign in, and confirm the dashboard and all `/settings/*` pages load without API errors.
5. Run a fresh agent install against the deployed backend and confirm a monitored server connects successfully.
