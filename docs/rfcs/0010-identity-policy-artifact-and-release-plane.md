# RFC 0010: Identity, Policy, Artifact, and Release Plane

## Status
Accepted baseline.

## Summary
This RFC closes the remaining enterprise plane gaps after the Postgres and durable worker work: normalized identity providers, external policy adapter boundary, encrypted secret provider baseline, remote object-store adapter boundary, immutable webhook replay payloads, and release supply-chain automation.

## Decisions

1. Identity is normalized behind `internal/identity.Provider`; handlers and services receive `model.CurrentUser` instead of provider-specific OIDC/SAML claims.
2. Policy uses `policy.Engine`. Built-in RBAC is the deterministic fallback; `CommandEngine` is the OPA/Rego-compatible external adapter boundary.
3. Secrets are references, not raw manifest env. `EncryptedProvider` stores AES-GCM ciphertext and is compatible with a future Postgres/Vault/KMS provider.
4. Artifacts use `artifact.Store`. `HTTPObjectStore` provides a zero-dependency remote gateway contract; direct S3/MinIO SDK support can be added behind the same interface.
5. Webhook retries must replay an immutable payload, not reconstruct a lossy event shell.
6. Release artifacts require checksums; enterprise releases should add cosign, SBOM, and provenance.

## Non-goals
This RFC does not vendor a Postgres driver or OPA runtime. Those are release-profile dependencies and should be added deliberately with integration tests.
