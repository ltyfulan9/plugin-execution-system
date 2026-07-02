# Security Specification

## Security Goals

The platform must protect the host process, tenant data, secrets, audit trail and plugin execution boundary.

## Runtime Security

Runtime contract must include:

- command or image declaration.
- timeout.
- stdout/stderr limit.
- output sanitization.
- network policy.
- env allowlist.
- mount allowlist.
- CPU/memory/pids limits.
- secret reference injection.

Process runner is allowed for local/dev and compatibility. Enterprise runtime should prefer container runner or remote isolated runner.

## Manifest Security

Plugin manifest should include:

- checksum.
- signature.
- provenance.
- declared capabilities.
- declared permissions.
- declared resources.
- runtime type.
- network policy.
- secret references.

The platform should reject plugins whose declared permissions exceed policy decisions.

## Secret Handling

Secrets must not be passed as arbitrary inline environment variables.

Required direction:

- plugins declare secret references.
- policy engine decides whether the plugin may use the reference.
- secret provider resolves the reference at runtime.
- audit records the reference, never the secret value.

## Tenant and Project Scope

Production implementations must enforce:

- tenant_id on core entities.
- project_id on project-scoped entities.
- scope-aware repository queries.
- no cross-tenant result, webhook, audit, plugin or execution leakage.

## Audit

Audit records should include:

- actor.
- tenant.
- project.
- resource.
- action.
- decision.
- reason.
- request_id.
- trace_id.
- plugin_digest.
- input_hash.
- result_hash.

Management operations, policy denial, plugin execution, permission elevation and security-sensitive failures must be audited.

## Webhook Security

Webhook delivery uses HMAC-SHA256 signature.

Headers:

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

SSRF baseline should reject private, loopback, link-local and unspecified targets unless explicitly allowed in test mode.

## Release Security

Release pipeline should provide:

- checksums.
- SBOM.
- vulnerability scan.
- signed artifacts.
- container image digest.
- provenance / attestation.

Current repository includes baseline scripts and workflow placeholders. Production release must wire them into a real trusted CI environment.
