# RFC 0004: Enterprise Metadata Store and Durable Execution

## Status

Accepted as the enterprise architecture direction for v0.7+.

## Decision

Production PES deployments use Postgres or an HA-compatible metadata store as source of truth. Local JSON is not a production implementation.

Core rules:

1. Every core entity is tenant/project scoped.
2. Current task state is stored in `executions`.
3. Historical state changes are stored in `task_events`.
4. Attempts are stored in `task_attempts`.
5. Results/artifacts/logs are stored outside the current-state row.
6. Idempotency is enforced with a unique constraint.
7. Work is leased using a durable queue with heartbeat and visibility timeout.
8. Audit and business events are separate.

## Queue baseline

The first production queue target is Postgres SKIP LOCKED because it keeps the deployment small and is suitable for moderate scale. The interface also allows Temporal, Kafka, or NATS JetStream later.

## Non-goals

- In-memory queue as production source of truth.
- JSON file metadata as production source of truth.
- SQLite as production default.
