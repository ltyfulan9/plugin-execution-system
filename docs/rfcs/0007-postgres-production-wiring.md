# RFC 0007: Postgres Production Wiring

Status: accepted

## Decision

The production server path uses Postgres-backed repositories, the Postgres metadata store, the Postgres durable queue, and the durable worker pool. `local-json` remains a dev/test adapter only.

## Why

Enterprise execution cannot rely on split writes, in-memory queues, or scan-based recovery. Task creation must atomically write:

1. current execution state,
2. append-only task events,
3. append-only audit record,
4. durable queue row.

## Implementation

`cmd/server` now selects by `METADATA_STORE`:

- `postgres`: production path using `repository.NewPostgresRepositories`, `postgres.MetadataStore`, `queue.PostgresQueue`, and `worker.DurableWorkerPool`.
- `local-json`: explicitly gated dev/test path.

`EnterpriseExecutionService` is the production execution service. It creates executions through `MetadataStore.CreateExecutionAndEnqueue`, so idempotency is enforced by the Postgres unique index and task queue state is committed with execution state.

`MetadataExecutionHandler` is the durable worker handler. It handles leased queue items and writes attempt records, state events, results, final status, and audit entries through metadata-store transactions.

## Non-goals

This RFC does not vendor a Postgres driver. Deployments must register a real `database/sql` Postgres driver at the application boundary, for example pgx stdlib or lib/pq. The platform must not silently fall back to local-json when Postgres is unavailable.

## Validation

The codebase must pass:

```bash
go test ./...
go test -race ./...
```

Production integration validation requires a real Postgres service and `POSTGRES_DSN`.
