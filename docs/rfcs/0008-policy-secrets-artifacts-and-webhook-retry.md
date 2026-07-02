# RFC 0008: Policy, Secrets, Artifacts, and Webhook Retry Baseline

Status: accepted baseline

## Motivation

Earlier versions had production execution wiring, but several enterprise planes were still too implicit:

- authorization was mostly role checks in service methods;
- secret handling was only a contract, not an executable resolver;
- large outputs had no object-store abstraction;
- webhook failure handling was single-shot;
- webhook target validation did not prevent common SSRF mistakes.

v11 introduces baseline implementations without making local/dev adapters the production model.

## Decisions

### Policy

A `policy.Engine` interface is the stable seam for OPA/Rego or external PDP integrations. The built-in `RBACEngine` is intentionally small but production-oriented enough to enforce:

- actor identity presence;
- tenant/project scope matching before role checks;
- role/action permissions;
- explicit deny reasons.

Enterprise execution create/read/cancel calls the policy engine.

### Secrets

Plugins declare `runtime.secretRefs`; they do not receive raw secret values in the manifest. Runtime resolves references at execution time with tenant/project scope. v11 allows secret env injection only into container runtime. Process runtime remains compatibility/dev and refuses inline env.

### Artifacts

`internal/artifact.Store` defines object-store semantics. The local adapter is dev/test only. Production deployments should use S3, GCS, Azure Blob, MinIO, or an equivalent HA object store. Metadata rows should store URI, digest, and size.

### Webhooks

Webhook deliveries now support retry metadata and DLQ state. Failed deliveries receive exponential backoff. Exhausted attempts move to DLQ. A dedicated durable retry scheduler is left for a later RFC.

### SSRF baseline

Webhook creation rejects literal localhost, loopback, private, link-local, and unspecified IP hosts by default. DNS-based rebinding and egress proxy policy require a dedicated enterprise network policy layer.

## Non-goals

- Full OPA adapter.
- Vault/KMS provider.
- S3/MinIO implementation.
- Durable webhook scheduler.
- DNS rebinding-proof webhook egress proxy.

## Follow-ups

- RFC for production pgx driver and Postgres integration CI.
- RFC for object storage adapters.
- RFC for OPA/Rego integration.
- RFC for Vault/KMS secret providers.
