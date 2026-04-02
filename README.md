# log-mcp

Read-only [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server for log analysis. The server communicates over stdio and exposes tools for inspecting log data without mutating sources.

## Status

Early development. The public API and tool surface are not stable.

## Architecture

log-mcp is a **stdio MCP server**: the host launches the binary and speaks [Model Context Protocol](https://modelcontextprotocol.io/) over newline-delimited **JSON-RPC 2.0** on stdin/stdout. Logging and diagnostics must go to **stderr** only so stdout stays a clean protocol channel.

**Full architecture** (packages, log sources, tools, environment variables, security limits, and data flow) lives in **[docs/architecture.md](docs/architecture.md)**.

| Layer | Package / path | Responsibility |
|-------|----------------|----------------|
| Entry | `cmd/log-mcp` | Process lifetime, signal handling, `server.Run`. |
| Transport & session | `github.com/modelcontextprotocol/go-sdk/mcp` | `StdioTransport`, JSON-RPC framing, MCP session lifecycle. |
| Composition | `internal/server` | Builds `mcp.Server`, applies the default **tool registry**, exposes `Run` for stdio. |
| Wire model | `internal/server` (`Request` / `Response`) | Typed JSON-RPC envelopes for tests and future custom transports. |
| Tools | `internal/tools` | **Registry** (`Register` + `Apply`) and MCP tools (health, logs, analysis, LLM helpers, …). |
| Metadata | `internal/buildinfo` | Server `Name` / `Version` surfaced by `version_info` and MCP `Implementation`. |

**Tool registry:** installers are registered on a `tools.Registry` and applied in order to `mcp.Server` via `mcp.AddTool`, keeping tool wiring in one place as the surface grows.

**Request/response:** MCP maps cleanly to JSON-RPC: clients send `tools/call` with `params`; handlers use the SDK’s typed `ToolHandlerFor[In, Out]` so inputs/outputs are validated and serialized per the spec.

## Layout

| Path | Purpose |
|------|---------|
| `cmd/log-mcp` | MCP server entrypoint (stdio binary) |
| `internal/server` | MCP server wiring, JSON-RPC wire types, stdio `Run` |
| `internal/tools` | Tool registry and tool implementations |
| `internal/` | Other private application code |
| `pkg/` | Library code intended for import by other modules |
| `docs/` | Project documentation |
| `test/` | Shared test fixtures and integration helpers |

## Requirements

- Go 1.22 or later

## Build

```bash
go build -o log-mcp ./cmd/log-mcp
```

## License

Apache License 2.0. See [LICENSE](LICENSE).

Copyright 2026 Ioannis Polyzos.
