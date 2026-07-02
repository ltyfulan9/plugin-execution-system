# RFC 0006: Production mode, trace propagation and structured logging

## Problem

Earlier versions could still accidentally present local JSON storage as if it were a valid production route. That violates the enterprise platform contract: metadata must be durable, transactional, tenant scoped, and recoverable.

The HTTP logs were also plain strings and did not carry a trace identifier that can be propagated from API to workers and plugin runtime.

## Decision

1. `APP_MODE=production` is the default.
2. `METADATA_STORE=local-json` is forbidden in production.
3. Local JSON requires `APP_MODE=dev|test` and `ALLOW_LOCAL_JSON_STORE=true`.
4. Postgres mode requires `POSTGRES_DSN` and a registered Postgres `database/sql` driver.
5. Server startup must fail closed if production metadata is not configured.
6. HTTP logs are emitted as JSON records.
7. Trace id is accepted from `Trace-ID` or W3C `traceparent`; otherwise generated.

## Non-goals

This RFC does not finish the full Postgres repository wiring. That is RFC 0007 scope.

## Follow-up

- RFC 0007: repository interface extraction and Postgres repository adapters.
- RFC 0008: Postgres integration test harness.
- RFC 0009: OpenTelemetry SDK integration.
