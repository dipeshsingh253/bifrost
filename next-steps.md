1. Run the latest generated Docker install command for a fresh system onboarding and verify it now includes `--pid host`, `--uts host`, `-v /:/hostfs:ro`, and `BIFROST_HOST_ROOT=/hostfs`.
2. Start the Docker-installed agent and confirm the connected server shows the host machine hostname, uptime, CPU/memory usage, and disk usage instead of container-local values.
3. Open the admin viewer access page after revoking one invite and letting another expire, then confirm only still-pending invites appear in the pending list and count.
4. Apply the migration chain to a database that already contains metric, log, and event rows, then confirm the rows remain available after the UUID migration step.
