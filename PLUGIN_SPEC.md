# Plugin Specification

This document defines the plugin contract for Plugin Execution System.

## Runtime Contract

Current verified local runtime:

- runtime type: `process`
- protocol: `stdio-json`
- plugin language: any language that can read stdin and write stdout

Reserved enterprise runtimes:

- `container`
- `wasm`
- `remote`

Process runtime is kept for local/dev and compatibility. Container runtime is the preferred enterprise direction.

## Directory Layout

```text
plugins/
  echo/
    manifest.json
    run.py
```

Each plugin directory must contain one `manifest.json`.

## Preferred Manifest Format

New plugins should use `plugin.exec/v1`.

```json
{
  "apiVersion": "plugin.exec/v1",
  "kind": "Plugin",
  "metadata": {
    "name": "echo",
    "displayName": "Echo",
    "version": "1.0.0",
    "description": "Return the input payload unchanged.",
    "license": "Apache-2.0"
  },
  "runtime": {
    "type": "process",
    "protocol": "stdio-json",
    "entrypoint": "python3",
    "args": ["run.py"],
    "timeoutSeconds": 5
  },
  "compat": {
    "coreApi": "v1",
    "protocolVersion": 1
  },
  "capabilities": ["task.execute"],
  "permissions": {
    "network": "none",
    "fs": []
  },
  "resources": {
    "memory": "64Mi",
    "pids": 8
  },
  "security": {
    "checksum": "dev-local"
  }
}
```

## Compatibility Manifest

Legacy flat manifests are accepted for compatibility:

```json
{
  "name": "echo",
  "version": "1.0.0",
  "entry_type": "process",
  "command": "python3",
  "args": ["run.py"],
  "timeout_seconds": 5
}
```

New plugins should not use the flat format unless they are only for local compatibility tests.

## Host-to-plugin Request

The host writes one JSON object to stdin:

```json
{
  "execution_id": "exec_xxx",
  "plugin_id": "plugin_xxx",
  "request_id": "req_xxx",
  "input": {
    "text": "hello world"
  },
  "metadata": {
    "plugin_name": "echo",
    "plugin_version": "1.0.0"
  }
}
```

## Plugin-to-host Response

Success:

```json
{
  "success": true,
  "data": {
    "echo": "hello"
  },
  "metrics": {
    "items": 1
  }
}
```

Failure:

```json
{
  "success": false,
  "error": "controlled failure"
}
```

## Result Mapping

- Valid JSON and `success: true` -> `Success`.
- Valid JSON and `success: false` -> `Failed`.
- Timeout -> `Timeout`.
- Non-JSON stdout -> `InvalidOutput`.
- Command/path/output-size/process error -> `RuntimeError`.

## stderr Rules

stderr is diagnostics only. It is captured, truncated, sanitized and exposed only as preview text. It must not contain secrets.

## Runtime Security Rules

Process runtime controls:

- bare commands must be allowlisted by host config.
- absolute command paths are rejected.
- relative command paths cannot escape plugin directory.
- command and args cannot contain NUL bytes.
- timeout is capped by host config.
- stdout/stderr are size-limited.
- stdout/stderr previews are sanitized.
- network access declarations are rejected by policy unless supported by the selected runner.
- ordinary inline env must not be used for secrets.

Container runtime contract:

- image should be pinned by digest.
- network defaults to none.
- mounts default to read-only.
- privileged mode is forbidden.
- memory/cpu/pids limits should be enforced.
- secret injection must come from secret references.

## Checksum and Signature

Release-grade manifests should provide a real entrypoint checksum:

```json
"security": {
  "checksum": "sha256:<64 hex chars>"
}
```

For interpreter plugins such as `python3 run.py`, the platform verifies the first relative script argument, for example `run.py`. For relative binary entrypoints such as `./plugin`, the platform verifies that binary.

Release plugins may also provide an Ed25519 signature:

```json
"security": {
  "checksum": "sha256:<64 hex characters>",
  "signature": "ed25519:<base64 signature over entrypoint bytes>"
}
```

Trusted public keys are supplied through platform configuration. Local examples may use `dev-local`, but that is not a production-grade verification policy.

## Secret References

Plugins must not request sensitive values through ordinary inline environment variables.

Use secret references:

```json
{
  "runtime": {
    "type": "container",
    "protocol": "stdio-json",
    "image": "example/plugin@sha256:...",
    "secretRefs": {
      "OPENAI_API_KEY": "secret://tenant/project/openai"
    }
  }
}
```

The runtime resolves secret references immediately before execution and injects them only according to runtime policy. Audit records should contain the secret reference, not the secret value.

## Artifact Contract

Large plugin outputs, logs and files should become artifacts in object storage. Database result rows should store metadata, URI, digest and size rather than large opaque blobs.

## Compatibility Policy

Within `plugin.exec/v1`, adding optional fields is allowed. Removing fields, changing field meaning or changing runtime semantics requires a future major plugin API version.
