# Contributing

Thanks for helping improve Plugin Execution System.

## Development loop

Run these before opening a PR:

```bash
gofmt -w $(find . -name '*.go')
go test ./...
go test -race ./...
```

## Design rules

- `handler` only parses HTTP and returns responses.
- `service` owns business rules, state machines, idempotency, and permission checks.
- `repository` only reads/writes storage.
- `model` only defines data structures and constants.
- `runtime` only runs plugins and isolates failures.
- `worker` only consumes queued execution IDs and calls services.

## Commit sign-off

This project prefers DCO-style sign-off for external contributions:

```text
Signed-off-by: Your Name <you@example.com>
```

## Pull request checklist

- Add or update tests for behavior changes.
- Update docs when changing API, manifest, runtime, or security behavior.
- Keep backward compatibility for legacy manifests unless the change is explicitly marked breaking.
- Do not weaken command/path/output/security checks without a security review.
