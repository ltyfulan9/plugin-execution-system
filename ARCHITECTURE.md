# Architecture

## Overview

Plugin Execution System is organized around a clear execution loop:

```text
Plugin manifest -> Registry -> Execution -> Queue -> Worker -> Runtime -> Result -> Summary -> Event/Audit -> API/UI
```

The platform owns scheduling, state, isolation, observability and audit. Plugins only provide bounded capability through a declared runtime protocol.

## Layering

```text
cmd/server      startup and dependency wiring
router          route registration
middleware      auth, request id, trace id, logging, recovery
handler         HTTP parameter parsing and response mapping
service         business rules, state machines, idempotency, policy decisions
repository      data access abstractions and adapters
worker          background execution and retry orchestration
runtime         plugin runner abstraction and execution isolation
model           structs, enums, states, scope
response        unified JSON response and error codes
```

Rules:

- handler must not write business rules.
- service must not manually write HTTP responses.
- repository must not decide business state transitions.
- runtime must not know HTTP details.
- plugins must not depend on internal Go packages.

## Plugin Model

Plugins are described by manifest files and executed through runtime runners.

Supported runner contract:

- process
- container
- wasm
- remote

Current closed-loop verification uses process plugins for portability. Production recommendation is container runner with strict policy gates.

## Execution Model

Execution current state and execution history are separated:

- `executions`: current task state.
- `execution_results`: plugin-level results.
- `execution_events`: state and lifecycle event log.
- `execution_attempts`: worker attempt records.
- `audit_logs`: security and administrative audit records.

This separation makes the system explainable. A task can be reconstructed through events, attempts and audit records.

## State Changes

Important state changes should emit event records:

- ExecutionCreated
- ExecutionQueued
- ExecutionStarted
- PluginStarted
- PluginFinished
- ExecutionFinished
- ExecutionFailed
- ExecutionCanceled
- ExecutionRecovered

## Idempotency

Execution creation supports `Idempotency-Key`.

Local/dev mode uses repository-level idempotency logic. Production route must enforce idempotency with a metadata store unique constraint.

## Scope Model

Core entities are designed with tenant/project scope:

```text
tenant_id
project_id
```

Any production repository implementation must enforce scope in queries and mutations.

## Runtime Isolation

Runtime contract includes:

- command or image declaration.
- env allowlist.
- secret references.
- mount policy.
- network policy.
- CPU/memory/pids limits.
- timeout.
- stdout/stderr size limit.
- output sanitization.

Process runner is kept for local/dev and compatibility. Container runner is the preferred enterprise direction.

## Security Plane

The platform contains boundaries for:

- Identity provider: local bootstrap, OIDC, SAML, mTLS.
- Policy engine: built-in RBAC and external command/OPA-style engine boundary.
- Secret provider: reference-based secret resolution.
- Artifact store: object-store style result/log storage.
- Audit plane: append-style security and operation records.

## Observability

The system exposes:

- `/livez`
- `/readyz`
- `/workerz`
- `/dependencyz`
- `/metrics`
- `/debug/vars`
- request_id
- trace_id
- structured log baseline

## Storage Direction

Local/dev mode uses local-json to make the project easy to verify without external services.

Production direction is Postgres or an HA metadata store, with durable queue semantics and transactionally recorded state/event/audit changes.

## Verification

The canonical acceptance proof is:

```bash
make verify
```

It validates the end-to-end execution loop rather than only compiling code.
