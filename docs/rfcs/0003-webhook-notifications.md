# RFC 0003: Webhook Notifications

Status: implemented baseline in v6

## Context

PES originally exposed execution state through REST, SSE, and audit logs. That is enough for demos, but an open-source platform needs an outbound integration surface so CI systems, ticket systems, dashboards, and automation agents can react to execution lifecycle events.

## Decision

PES adds admin-managed webhook endpoints. Each endpoint declares a URL, status, event filters, and a shared secret. When the execution event bus emits an event, PES delivers matching events to enabled endpoints.

## Delivery shape

PES sends an HTTP `POST` with JSON body:

```json
{
  "type": "ExecutionFinished",
  "event": {},
  "sent_at": "2026-07-01T00:00:00Z"
}
```

Headers:

```text
X-PES-Event: ExecutionFinished
X-PES-Delivery: whd_xxx
X-PES-Timestamp: 1782910000
X-PES-Signature: sha256=<hmac>
```

The signature is HMAC-SHA256 over `<timestamp>.<raw_body>`.

## Current limitations

The v6 baseline stores failed deliveries but does not retry them. A production implementation should add retry scheduling, backoff, dead-letter state, and encrypted secret storage.
