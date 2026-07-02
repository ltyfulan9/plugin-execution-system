# Security Policy

## Supported versions

This repository is currently pre-1.0. Security fixes are applied to the latest `main` branch and the most recent tagged minor release once releases begin.

## Reporting a vulnerability

Please do not open a public issue for a suspected vulnerability. Report privately to the maintainers listed in `MAINTAINERS.md`.

Expected response target:

- Initial acknowledgement: within 72 hours.
- Triage result or mitigation plan: within 14 days.
- Patch or advisory target: within 30 days for confirmed high-impact issues.

## Security boundaries

The default runtime is process isolation plus command allowlist, timeout, output limits, request-scoped input, and audit logging. It is not a full sandbox. Running untrusted third-party plugins should use an external sandbox such as a container runner with seccomp/rootless/network isolation before production use.

## In scope

- Plugin command/path escape.
- Execution state corruption or cross-user access.
- Token bypass or privilege escalation.
- Unbounded stdout/stderr memory growth.
- ANSI/control-sequence log or UI injection.
- Idempotency bypass that causes duplicated execution.

## Out of scope for the current local demo runtime

- Malicious plugin code reading files that the OS user can already read.
- Kernel/container escape bugs outside this repository.
- Denial of service caused by intentionally giving plugins high host privileges.
