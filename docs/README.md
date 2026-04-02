# Documentation

**Authoritative detail** for **log-mcp** lives in this folder. The root
[README.md](../README.md) is a short overview and entry point; when the two
differ, treat **`docs/`** as the source of truth for behavior and configuration.

## In this directory

| Document | Contents |
|----------|----------|
| [architecture.md](architecture.md) | End-to-end design: MCP stdio model, **complete tool catalog**, all log-source environment variables, security limits (`LOG_MCP_SEC_*`), parsing and normalization, streaming pipeline, file/k8s/docker/CF backends. |
| [testing-and-ci.md](testing-and-ci.md) | Parser fixtures, **`go test` / `go vet`**, **`govulncheck`**, **`go mod tidy`** gate, lint jobs, workflow map, releases. |

## Repository guides (repo root)

These files are part of community health and contributor onboarding:

| File | Description |
|------|-------------|
| [README.md](../README.md) | Project summary, doc map, minimal MCP notes, layout, build, CI pointer. |
| [CONTRIBUTING.md](../CONTRIBUTING.md) | Branching, local commands, PR expectations. |
| [CODE_OF_CONDUCT.md](../CODE_OF_CONDUCT.md) | Contributor Covenant. |
| [SECURITY.md](../SECURITY.md) | Responsible disclosure. |
| [LICENSE](../LICENSE) | Apache License 2.0. |

## Updating docs

When you change tools, env vars, security limits, or pipeline behavior, update
[architecture.md](architecture.md) and, if checks or fixtures move,
[testing-and-ci.md](testing-and-ci.md). User-visible one-liners in the root
README should stay in sync or defer here with a link.
