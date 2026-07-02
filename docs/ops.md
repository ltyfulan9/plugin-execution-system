# Operations Guide

## Endpoints

- `/api/health`: liveness.
- `/api/ready`: readiness.
- `/debug/vars`: expvar metrics.
- `/openapi.json`: OpenAPI contract.

## Important metrics

- `pes_http_requests_total`
- `pes_http_status_total`
- `pes_task_submitted_total`
- `pes_task_recovered_total`
- `pes_task_completed_total`
- `pes_queue_full_total`
- `pes_idempotency_hits_total`
- `pes_plugin_started_total`
- `pes_plugin_completed_total`
- `pes_sandbox_denials_total`

## Restart behavior

This version uses JSON storage and an in-memory queue. On startup, persisted Pending and Queued executions are re-submitted to the queue. Persisted Running executions are marked Failed because the local process runner cannot prove whether the child process survived a host restart.

Future versions should replace this with task attempts, leases, and heartbeat-based recovery.
