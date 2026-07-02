# API Specification

Canonical API namespace: `/api/v1`.

Compatibility routes may exist for older clients, but new behavior must be defined under `/api/v1`.

## Response Format

Success:

```json
{
  "code": "OK",
  "message": "success",
  "data": {},
  "request_id": "req_xxx"
}
```

Error:

```json
{
  "code": "IDEMPOTENCY_CONFLICT",
  "message": "idempotency conflict",
  "error": {
    "type": "business_error",
    "detail": "same idempotency key was used with different input"
  },
  "request_id": "req_xxx"
}
```

## Stable Error Codes

- `OK`
- `INVALID_ARGUMENT`
- `UNAUTHORIZED`
- `FORBIDDEN`
- `NOT_FOUND`
- `PLUGIN_NOT_FOUND`
- `PLUGIN_DISABLED`
- `PLUGIN_STATE_INVALID`
- `MANIFEST_INVALID`
- `EXECUTION_NOT_FOUND`
- `EXECUTION_STATE_INVALID`
- `IDEMPOTENCY_CONFLICT`
- `QUEUE_FULL`
- `PLUGIN_RUNTIME_ERROR`
- `PLUGIN_TIMEOUT`
- `PLUGIN_INVALID_OUTPUT`
- `POLICY_DENIED`
- `INTERNAL_ERROR`

## Health APIs

```text
GET /livez
GET /readyz
GET /workerz
GET /dependencyz
GET /api/v1/health
GET /api/v1/ready
GET /metrics
GET /debug/vars
```

## Auth APIs

```text
POST /api/v1/auth/login
GET  /api/v1/auth/me
```

Local/dev tokens:

```text
admin-token
demo-token
```

## Plugin APIs

```text
POST /api/v1/plugins/reload
GET  /api/v1/plugins?page=1&page_size=20
GET  /api/v1/plugins/{id}
POST /api/v1/plugins/{id}/enable
POST /api/v1/plugins/{id}/disable
```

Notes:

- reload、enable、disable 需要管理员权限。
- 普通用户可以查看插件列表和详情。
- 只有 Enabled 插件可以被执行。

## Execution APIs

```text
POST /api/v1/executions
GET  /api/v1/executions?page=1&page_size=20
GET  /api/v1/executions/{id}
POST /api/v1/executions/{id}/cancel
GET  /api/v1/executions/{id}/results
GET  /api/v1/executions/{id}/summary
GET  /api/v1/executions/{id}/events
GET  /api/v1/executions/{id}/events/stream
GET  /api/v1/executions/{id}/attempts
```

Create execution headers:

```text
Authorization: Bearer demo-token
Idempotency-Key: user-generated-key
```

Create execution body:

```json
{
  "plugin_ids": ["plugin_xxx"],
  "input": {
    "text": "hello plugin system"
  }
}
```

Idempotency semantics:

- Same scope + same user + same key + same plugin set + same input hash returns the existing execution.
- Same key with different input or plugin set returns `IDEMPOTENCY_CONFLICT` with HTTP 409.

## Audit APIs

```text
GET /api/v1/audit/logs?page=1&page_size=20
GET /api/v1/audit/executions/{id}
GET /api/v1/audit/plugins/{id}
```

Audit queries require admin permission.

## Webhook APIs

```text
POST   /api/v1/webhooks
GET    /api/v1/webhooks?page=1&page_size=20
GET    /api/v1/webhooks/{id}
POST   /api/v1/webhooks/{id}/enable
POST   /api/v1/webhooks/{id}/disable
DELETE /api/v1/webhooks/{id}
GET    /api/v1/webhooks/{id}/deliveries
GET    /api/v1/webhooks/deliveries
```

Webhook delivery security headers:

```text
X-PES-Event
X-PES-Delivery
X-PES-Timestamp
X-PES-Signature
```

Signature payload:

```text
<timestamp>.<raw_body>
```

Signature algorithm: HMAC-SHA256.

## Pagination

List APIs must support:

```text
page
page_size
```

Default page size should be conservative. API responses should include total count and items when applicable.

## OpenAPI

OpenAPI document is available at:

```text
GET /openapi.json
```

The source file is `docs/openapi.json`.
