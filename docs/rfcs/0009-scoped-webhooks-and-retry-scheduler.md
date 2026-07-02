# RFC 0009: Scoped Webhooks and Retry Scheduler

## Status

Accepted for v12 baseline.

## Problem

Webhook endpoints are integration boundaries. In a multi-tenant platform, a webhook endpoint from one tenant/project must never receive execution events from another tenant/project. Retry also cannot be an ad-hoc manual helper; failed deliveries need a background reliability loop with retry/backoff/DLQ semantics.

## Decision

1. Webhook management APIs are scope-aware.
   - Non-super-admin callers can only list/get/update/delete webhooks in their own tenant/project.
   - Super admins can operate across scopes.

2. Webhook dispatch is scope-aware.
   - An execution event is delivered only to enabled webhook endpoints in the same tenant/project.

3. Retry scheduler becomes part of server wiring.
   - `WebhookRetryScheduler` periodically calls `RetryFailedDeliveries`.
   - Failed deliveries continue to use exponential backoff and DLQ decisions already owned by `WebhookService`.

4. Observability is mandatory.
   - Retry successes increment `pes_webhook_retry_total`.
   - Retry loop errors increment `pes_webhook_retry_error_total`.

## Non-goals

- This RFC does not add a separate Postgres delivery queue yet.
- This RFC does not add encrypted webhook secrets yet.
- This RFC does not add manual delivery replay API yet.

## Follow-up

- Persist immutable webhook payload artifacts for exact replay.
- Add webhook secret encryption and rotation.
- Add SSRF DNS rebinding protection at delivery time, not only create time.
- Add manual replay endpoint with policy/audit checks.
