# Client contract

How **log-mcp** behaves at the MCP boundary. This server uses the official Go SDK
[`github.com/modelcontextprotocol/go-sdk/mcp`](https://github.com/modelcontextprotocol/go-sdk)
with **`StdioTransport`**: the MCP session is **JSON-RPC 2.0** over **stdin/stdout**,
not a custom line-delimited tool protocol.

For tool names, arguments, and structured results, see [tools.md](tools.md).
For environment-driven backends and limits, see [architecture.md](architecture.md).

## Transport and channels

| Channel | Use |
|---------|-----|
| **stdin** | MCP client → server messages. |
| **stdout** | MCP server → client messages **only**. Must remain valid JSON-RPC for the client. |
| **stderr** | Process diagnostics (for example `log.Fatal` from `cmd/log-mcp`). Hosts must not parse stderr as MCP. |

Do not print debug text to **stdout**; it will corrupt the MCP stream.

## Tool call success and failure

Handlers registered with `mcp.AddTool` return a structured **result** type on success.
On failure they return a Go **`error`**. The SDK maps that to an MCP tool result with
**`isError: true`** and a textual description (the error’s message).

There is **no** separate stable `error.code` enum in this repository (unlike a custom
stdio JSON server). Clients should treat tool failures as **opaque strings** for display
and logging, and rely on **JSON Schema** (from struct tags) for input validation where
the host exposes it.

## Typical error message patterns

These strings are **not** a versioned API; they may change. They are listed to help
operators and client authors grep logs.

| Situation | Example message shape |
|-----------|------------------------|
| No log source configured | `log access not configured: set LOG_MCP_ROOT to a readable file or directory …` |
| Missing `stream_id` | `stream_id is required` |
| Missing `keyword` (search) | `keyword is required for search_logs` |
| Empty filters (filter_logs) | `filter_logs requires at least one of: since, until, level, keyword` |
| Bad RFC3339 time | `invalid RFC3339 time "…": …` |
| Missing correlation field / value | `field is required`, `value is required`, `trace_id is required` |
| LLM tools | `structured_summary is required`, `provide exactly one of error_group or repeated_burst`, validation errors from [`internal/llmtools`](../internal/llmtools/) |
| Source I/O / path policy | Messages from [`internal/source`](../internal/source/) or [`internal/security`](../internal/security/) (path outside roots, command argv rejection, etc.) |

## Session lifecycle

- The process runs until the client disconnects, the context is cancelled (signals in
  `cmd/log-mcp`), or a fatal error on startup.
- **Protocol-level** parse errors are handled by the MCP SDK and host; consult the
  [MCP specification](https://modelcontextprotocol.io/) for JSON-RPC error objects at
  the transport layer.

## Related docs

- [usage.md](usage.md) — running the binary and configuring hosts.
- [troubleshooting.md](troubleshooting.md) — common failures.
- [tools.md](tools.md) — per-tool inputs and outputs.
