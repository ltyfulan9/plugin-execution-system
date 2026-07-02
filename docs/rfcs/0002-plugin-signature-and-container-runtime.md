# RFC 0002: Plugin Signatures and Optional Container Runtime

## Status

Implemented baseline in v5.

## Motivation

A globally useful plugin platform needs two safety upgrades beyond local process plugins:

1. Release-grade plugin integrity using checksums and public-key signatures.
2. A path toward stronger runtime isolation through container execution.

## Design

Process plugins may declare:

```json
{
  "security": {
    "checksum": "sha256:<entrypoint-sha256>",
    "signature": "ed25519:<base64-signature-over-entrypoint-bytes>"
  }
}
```

The core is configured with trusted public keys through `PLUGIN_TRUSTED_PUBLIC_KEYS`.
Multiple keys are comma-separated and use `ed25519:<base64-raw-public-key>`.

Container plugins declare:

```json
{
  "runtime": {
    "type": "container",
    "protocol": "stdio-json",
    "image": "ghcr.io/org/plugin:1.0.0",
    "args": ["run"],
    "timeoutSeconds": 10
  },
  "permissions": {"network": "none"},
  "resources": {"memory": "128Mi", "cpu": "0.5", "pids": 64}
}
```

Container execution is disabled by default and must be enabled with
`PLUGIN_CONTAINER_RUNTIME_ENABLED=true`.

## Non-goals

- This is not yet Sigstore or SLSA attestation verification.
- This is not yet Kubernetes-native execution.
- Docker availability is intentionally optional.
