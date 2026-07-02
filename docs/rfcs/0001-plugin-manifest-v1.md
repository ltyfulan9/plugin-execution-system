# RFC 0001: plugin.exec/v1 Manifest

Status: accepted

## Decision

PES supports `plugin.exec/v1` manifests with Kubernetes-like `apiVersion`, `kind`, `metadata`, `runtime`, `compat`, `capabilities`, `permissions`, `resources`, and `security` sections.

## Rationale

The manifest separates plugin identity, runtime protocol, compatibility, permission intent, resource intent, and supply-chain metadata. This lets the core keep a stable registry while changing runtime implementations over time.

## Compatibility

The legacy flat manifest is still accepted. New plugins should use `plugin.exec/v1`.

## Security

Release-grade plugins should set `security.checksum` to `sha256:<64 hex chars>` for the local entrypoint file. Development placeholders such as `dev-local` are allowed but are not treated as verified release checksums.
