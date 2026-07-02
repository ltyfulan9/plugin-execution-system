# Protocol Specification

## Current stable protocol: stdio-json v1

The host starts a plugin process and writes one JSON object to stdin.

Request fields:

- `execution_id`
- `plugin_id`
- `request_id`
- `input`
- `metadata.plugin_name`
- `metadata.plugin_version`

The plugin writes one JSON object to stdout.

Response fields:

- `success`: boolean
- `data`: object, required when success is true
- `error`: string, recommended when success is false
- `metrics`: object, optional plugin-specific metrics

stderr is reserved for diagnostics and is captured as a sanitized preview. It must not contain secrets.

## Future protocol: gRPC exec.v1

Reserved service shape:

- `GetMetadata`
- `GetCapabilities`
- `Health`
- `Execute`
- `Cancel`
- `StreamLogs`

The current codebase keeps this document so the project can move from stdio-json to generated SDKs without changing product semantics.
