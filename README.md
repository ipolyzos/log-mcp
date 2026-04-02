# log-mcp

Read-only [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server for log analysis. The host launches the `log-mcp` binary; the process speaks MCP over **stdio** (JSON-RPC on stdin/stdout) and exposes tools for inspecting log data **without mutating** underlying sources.

## Status

Active development on the `develop` branch. The MCP tool surface and JSON shapes may still evolve; pin releases or digests for production hosts if you need stability.

## MCP server (current behavior)

### Transport and runtime

- **Protocol:** MCP over **stdio** using `github.com/modelcontextprotocol/go-sdk/mcp` (`StdioTransport`). **stdout** is reserved for MCP messages; anything diagnostic must go to **stderr** so the channel stays valid JSON-RPC.
- **Lifecycle:** `cmd/log-mcp` installs signal handling, then `server.Run` blocks until the client disconnects or the process is interrupted.
- **Server identity:** MCP `Implementation` and the `version_info` tool use [`internal/buildinfo`](internal/buildinfo/buildinfo.go) (`Name`, `Version`). Release binaries can stamp `Version` via `-ldflags` (see [`.github/workflows/release.yml`](.github/workflows/release.yml)).

### How tools are wired

[`internal/server.NewWithConfig`](internal/server/server.go) constructs `mcp.NewServer` and applies a small **registry** ([`internal/tools`](internal/tools/registry.go)) in a fixed order:

1. `RegisterHealthCheck` — always useful for orchestration probes.
2. `RegisterVersionInfo` — build metadata.
3. `RegisterLogTools` — log read/search/filter, summarization, correlation, and advanced analysis (see table below).
4. `RegisterLLMTools` — deterministic helpers that consume **structured JSON only** (no direct log reads).

Each installer calls `mcp.AddTool` with typed handlers so inputs/outputs are validated against JSON Schema metadata from struct tags.

### Log sources and `LogSource`

At startup, [`logSourceFromEnv`](internal/server/server.go) optionally builds a [`source.LogSource`](internal/source/source.go) from environment variables (`LOG_MCP_ROOT`, Kubernetes, Docker, Cloud Foundry allowlists — full detail in [docs/architecture.md](docs/architecture.md)). If **no** source is configured, `LogSource` is `nil`: **log-backed tools still register** but return a clear configuration error when invoked (so clients can call `health_check` / `version_info` without mounting logs). `incident_summary` and `explain_log_pattern` **do not** require a log source.

### Tool catalog

| Tool | Needs configured log source? | Role |
|------|-----------------------------|------|
| `health_check` | No | Liveness-style probe (`status: ok`). |
| `version_info` | No | Server name and version string. |
| `read_logs` | Yes | Sequential read with optional `max_lines`, time range, level, keyword; output lines are **redacted**. |
| `search_logs` | Yes | Keyword search over redacted lines with optional filters and `max_matches`. |
| `filter_logs` | Yes | Lines matching ANDed filters (time / level / keyword). |
| `summarize_errors` | Yes | Deterministic ERROR/FATAL grouping and ranking ([`internal/errsummary`](internal/errsummary/)). |
| `group_by_field` | Yes | Group lines by a JSON field (`trace_id`, `request_id`, or custom key). |
| `correlate_events` | Yes | Timeline of lines matching an exact field value. |
| `detect_anomalies` | Yes | Segment-level error spikes and repeated-error bursts ([`internal/loganalysis`](internal/loganalysis/)). |
| `extract_trace_flow` | Yes | Ordered steps for a single trace id. |
| `incident_summary` | No | Validates merged structured output from other tools; returns a brief for downstream LLMs. |
| `explain_log_pattern` | No | Interprets one structured error group or burst; **no** log I/O. |

Redaction and limits (`LOG_MCP_SEC_*`, argv allowlists, read-only file access) are documented in [docs/architecture.md](docs/architecture.md#security-limits-log_mcp_sec_).

### Pipelines and parsers

Log lines flow through a **streaming** pipeline ([`internal/pipeline`](internal/pipeline/pipeline.go)): source → parse ([`internal/parse`](internal/parse/) JSON/text/auto) → redaction ([`internal/security`](internal/security/redact.go)) → optional analysis. The design avoids loading entire files into memory before processing.

## Architecture and documentation

| Document | Contents |
|----------|----------|
| [docs/architecture.md](docs/architecture.md) | End-to-end design: env vars, sources, security, parsing, normalization, pipeline. |
| [docs/README.md](docs/README.md) | Index of documentation in this repo. |
| [CONTRIBUTING.md](CONTRIBUTING.md) | How to propose changes, build, and test. |
| [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) | Community expectations. |
| [SECURITY.md](SECURITY.md) | How to report security issues. |
| [LICENSE](LICENSE) | Apache License 2.0. |

For a concise package map, see the **Layout** table below and the component table in [docs/architecture.md](docs/architecture.md#major-components).

## Layout

| Path | Purpose |
|------|---------|
| `cmd/log-mcp` | MCP server entrypoint (stdio binary) |
| `internal/server` | MCP server wiring, JSON-RPC wire types, stdio `Run` |
| `internal/tools` | Tool registry and tool implementations |
| `internal/` | Sources, pipeline, security, analysis, helpers |
| `pkg/` | Library code intended for import by other modules |
| `docs/` | Project documentation |
| `test/` | Shared test fixtures and integration helpers |
| `.github/workflows/` | CI, docs checks, benchmarks, release builds |

## Requirements

- **Go 1.25.8** or newer (matches [`go.mod`](go.mod); `setup-go` in CI uses `go-version-file`).

## Build

```bash
go build -o log-mcp ./cmd/log-mcp
```

Run under any MCP-compatible host that can spawn the binary with stdio connected and the desired environment variables set for your log backends.

## Continuous integration

Pull requests to `main` / `develop` run Go vet, tests, `golangci-lint`, `gosec`, `govulncheck`, documentation lint + link checks, and (on relevant branches) benchmarks. See [`.github/workflows/`](.github/workflows/).

## Community

- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Contributing](CONTRIBUTING.md)
- [Security policy](SECURITY.md)
- [License](LICENSE) — Apache License 2.0

Copyright 2026 Ioannis Polyzos.
