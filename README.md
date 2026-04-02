# log-mcp

Read-only [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server for log analysis. The server communicates over stdio and exposes tools for inspecting log data without mutating sources.

## Status

Early development. The public API and tool surface are not stable.

## Layout

| Path | Purpose |
|------|---------|
| `cmd/log-mcp` | MCP server entrypoint (stdio binary) |
| `internal/` | Private application code |
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
