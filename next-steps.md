1. Generate a fresh system onboarding and confirm the emitted `config.yaml` includes `docker.include_all`, all four include/exclude lists, and `logs.max_lines_per_fetch`.
2. Run an agent with `docker.include_all: true` and verify all Docker projects/containers appear except names listed in `exclude_projects` and `exclude_containers`.
3. Change to `docker.include_all: false`, set one compose project in `include_projects` and one standalone container in `include_containers`, restart the agent, and verify only those targets appear.
4. Lower `logs.max_lines_per_fetch`, restart the agent on a new host or clear its saved log cursor, and confirm the first fetch only tails that configured number of lines per container.
