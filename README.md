# log-mcp

Read-only [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server for log analysis. A host runs the `log-mcp` binary; the process speaks MCP over **stdio** (JSON-RPC on stdin/stdout) and exposes tools to inspect log data **without mutating** underlying sources.

## Documentation

**Detailed, maintained documentation is in [`docs/`](docs/).** The sections below are a compact overview; for environment variables, the full tool catalog, security limits, parsing behavior, and CI, follow the links.

| Resource | Purpose |
|----------|---------|
| [**docs/README.md**](docs/README.md) | Documentation index and update guidance. |
| [**docs/architecture.md**](docs/architecture.md) | **System design:** MCP stdio, sources, security limits, parsing, normalization, pipeline. |
| [**docs/tools.md**](docs/tools.md) | **Tool reference:** parameters and JSON results for every MCP tool. |
| [**docs/client-contract.md**](docs/client-contract.md) | Stdio/MCP expectations and how errors surface. |
| [**docs/usage.md**](docs/usage.md) | Build, env vars, host configuration, workflows. |
| [**docs/development.md**](docs/development.md) | Contributor workflow and doc ownership. |
| [**docs/testing-and-ci.md**](docs/testing-and-ci.md) | Tests, fixtures, `govulncheck`, `go mod tidy` gate, CI workflows. |
| [**docs/releases.md**](docs/releases.md) | Tags, release builds, versioning. |
| [**docs/troubleshooting.md**](docs/troubleshooting.md) | Common failures and fixes. |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Branches, build/test commands, PR checklist. |
| [SECURITY.md](SECURITY.md) | Reporting vulnerabilities. |
| [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) | Community expectations. |
| [LICENSE](LICENSE) | Apache License 2.0. |

## Status

Active development on the `develop` branch. The MCP tool surface and JSON shapes may still evolve; pin releases or digests for production hosts if you need stability.

## MCP server (summary)

- **Transport:** MCP over **stdio** via `github.com/modelcontextprotocol/go-sdk/mcp` (`StdioTransport`). **stdout** is reserved for MCP JSON-RPC; diagnostics must use **stderr** so the channel stays valid.
- **Lifecycle:** `cmd/log-mcp` installs signal handling, then `server.Run` blocks until disconnect or interrupt.
- **Identity:** MCP `Implementation` and the `version_info` tool use [`internal/buildinfo`](internal/buildinfo/buildinfo.go). Release binaries can stamp `Version` with `-ldflags` (see [`.github/workflows/release.yml`](.github/workflows/release.yml)).
- **Registration:** [`internal/server.NewWithConfig`](internal/server/server.go) builds `mcp.NewServer` and applies [`internal/tools`](internal/tools/registry.go) in order: health → version → log tools → LLM-oriented helpers. See [**tools.md →**](docs/tools.md) (parameters and JSON) and [**architecture →**](docs/architecture.md#log-mcp-tools) (sources and env vars).
- **Log source:** Optional [`source.LogSource`](internal/source/source.go) from the environment (`LOG_MCP_ROOT`, Kubernetes, Docker, Cloud Foundry allowlists — [**full list →**](docs/architecture.md#log-source-abstraction)). If unset, log-backed tools still register but return a clear configuration error; `incident_summary` and `explain_log_pattern` do not need a source.

**Redaction and limits** (`LOG_MCP_SEC_*`, argv allowlists, read-only file access): [**security section →**](docs/architecture.md#security-limits-log_mcp_sec_).

## Repository layout

| Path | Purpose |
|------|---------|
| `cmd/log-mcp` | MCP server entrypoint (stdio binary). |
| `internal/server` | MCP session wiring, `Run`. |
| `internal/tools` | Tool registry and implementations. |
| `internal/source`, `internal/pipeline`, `internal/parse`, `internal/security`, … | Sources, streaming pipeline, parsers, redaction, analysis. |
| `docs/` | Architecture, tools reference, usage, CI, releases, troubleshooting. |
| `test/` | Pointers to shared fixtures (see [`test/fixtures/logs/README.md`](test/fixtures/logs/README.md)). |
| `.github/workflows/` | CI, docs checks, benchmarks, release builds. |

## Requirements

- **Go** version matching [`go.mod`](go.mod) (CI uses `actions/setup-go` with `go-version-file: go.mod`).

## Build

```bash
go build -o log-mcp ./cmd/log-mcp
```

Configure the environment for your backends (see [architecture.md](docs/architecture.md)), then run under any MCP host that connects stdio to the process.

## Continuous integration

Pull requests to `main` / `develop` run **go mod tidy** (clean tree check), **go vet**, tests, **golangci-lint**, **gosec**, **govulncheck**, documentation lint and link checks, and benchmarks on selected branches. Details: [**docs/testing-and-ci.md**](docs/testing-and-ci.md).

## Community

- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Contributing](CONTRIBUTING.md)
- [Security policy](SECURITY.md)
- [License](LICENSE) — Apache License 2.0

Copyright 2026 Ioannis Polyzos.
