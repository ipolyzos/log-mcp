# Documentation

**Authoritative detail** for **log-mcp** lives in this folder. The root
[README.md](../README.md) is a short overview and entry point; when the two
differ, treat **`docs/`** as the source of truth for behavior and configuration.

## Start here

| Document | Contents |
|----------|----------|
| [architecture.md](architecture.md) | Goals, package map, MCP stdio, **log sources** (file, k8s, docker, CF), **security limits** (`LOG_MCP_SEC_*`), parsing, normalization, streaming pipeline. |
| [tools.md](tools.md) | **Every MCP tool**: parameters, result JSON, defaults, caps. |
| [client-contract.md](client-contract.md) | Stdin/stdout rules, how tool errors surface (MCP + Go SDK), typical message patterns. |
| [usage.md](usage.md) | Build, environment variables, host config patterns, common workflows. |
| [development.md](development.md) | Branching, local commands, parser fixtures, **which doc to edit** for a given change. |
| [testing-and-ci.md](testing-and-ci.md) | `go test`, fixtures, **`govulncheck`**, **`go mod tidy`** gate, workflow list. |
| [releases.md](releases.md) | Branch flow, SemVer tags, **`release.yml`** artifacts, `-ldflags` version stamping. |
| [CHANGELOG.md](../CHANGELOG.md) | Keep a Changelog history (**`[Unreleased]`** + tagged releases). |
| [kubernetes-runbook.md](kubernetes-runbook.md) | **K8s** stream IDs, CRI-shaped lines, allowlisted namespaces, suggested MCP checks. |
| [troubleshooting.md](troubleshooting.md) | Missing log source, `stream_id`, RFC3339, `filter_logs` / JSON args, kubectl/docker/cf, stdout corruption. |

## Repository guides (repo root)

| File | Description |
|------|-------------|
| [README.md](../README.md) | Project summary, doc map, minimal MCP notes, layout, build. |
| [CONTRIBUTING.md](../CONTRIBUTING.md) | Branching, build/test, PR checklist. |
| [CODE_OF_CONDUCT.md](../CODE_OF_CONDUCT.md) | Contributor Covenant. |
| [SECURITY.md](../SECURITY.md) | Vulnerability reporting. |
| [LICENSE](../LICENSE) | Apache License 2.0. |

## Updating docs

Use [development.md](development.md#documentation-updates-same-pr-as-behavior-changes) as the checklist. In short: behavior or tool JSON changes require **tools.md** and usually **architecture.md**; transport or error semantics → **client-contract.md**; runbooks → **usage.md** / **troubleshooting.md**.

Markdown in the repository is linted and link-checked in CI.
