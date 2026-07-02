# Storage Migration Plan

## Current

JSON files under `data/` are the default storage. This keeps the project dependency-free and easy to run.

## Next: SQLite

Add an implementation behind existing repository interfaces:

- `users`
- `plugins`
- `plugin_registry`
- `executions`
- `execution_results`
- `audit_logs`

Use WAL mode for better single-node read concurrency. Keep JSON import/export for local development.

## Future: Postgres

When multiple API/worker instances are supported, move to Postgres and add:

- `task_attempts`
- `task_events`
- `idempotency_keys`
- `scheduler_leases`

The service layer should not need major changes because repository interfaces already isolate storage details.
