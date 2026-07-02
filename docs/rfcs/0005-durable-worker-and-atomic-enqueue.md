# RFC 0005: Durable Worker and Atomic Enqueue

## Status

Accepted for v8 baseline.

## Problem

A task platform cannot treat process memory as the source of truth. If API writes an execution row and then crashes before queue publish, the task is lost. If a worker crashes after leasing a task, the task must become visible again. If retry history is not durable, operators cannot explain execution behavior.

## Decision

PES production execution creation must use one metadata transaction that writes:

- `executions` current state
- `task_events` append-only event
- `audit_logs` append-only audit decision
- `task_queue` durable ready row

Workers must consume by lease, not by in-memory queue. The durable worker loop is:

1. `LeaseNext(worker_id, lease_duration)`
2. create/continue attempt
3. heartbeat while running
4. run isolated runner
5. `Ack` on success
6. `Nack` with retry/backoff on failure
7. DLQ after max attempts
8. reclaim expired leases periodically

## Non-goals

This RFC does not mandate Postgres as the only possible queue forever. Kafka, NATS JetStream, or Temporal adapters can implement the same queue/workflow contract later. The first production baseline is Postgres `FOR UPDATE SKIP LOCKED` because it keeps metadata and queue transactionality simple.
