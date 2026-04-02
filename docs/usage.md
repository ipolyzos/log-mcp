# Usage

How to run **log-mcp** and point an MCP **client** at it. This server speaks **MCP over
stdio** (JSON-RPC); it is **not** a custom NDJSON tool protocol.

See [client-contract.md](client-contract.md) for stdin/stdout rules and
[tools.md](tools.md) for tool parameters.

## Build

From the repository root:

```bash
go build -o log-mcp ./cmd/log-mcp
```

Use a Go toolchain matching [`go.mod`](../go.mod).

## Configure log backends (environment)

Set **at least one** of the following before starting the process (see
[architecture.md](architecture.md#log-mcp-tools) for full semantics):

| Variable | Enables |
|----------|---------|
| `LOG_MCP_ROOT` | File or directory roots (OS path-list separator for multiple). |
| `LOG_MCP_K8S_NAMESPACES` | Comma-separated namespace allowlist → `kubectl` logs. |
| `LOG_MCP_DOCKER_CONTAINERS` | Comma-separated container name allowlist → `docker logs`. |
| `LOG_MCP_CF_APPS` | Comma-separated Cloud Foundry app allowlist → `cf logs`. |
| `LOG_MCP_PIPE_FD` | Integer **≥ 3**: read piped log lines from that OS file descriptor; MCP stdio stays on 0–2. Use **`stream_id` `pipe://`**. See [Piped logs (shell)](#piped-logs-shell) below. |

Optional overrides: `LOG_MCP_K8S_KUBECTL`, `LOG_MCP_DOCKER_DOCKER`, `LOG_MCP_CF_CLI`.

Optional **security caps** (positive integers; defaults if unset):

| Variable | Role |
|----------|------|
| `LOG_MCP_SEC_MAX_OUTPUT_LINES` | Cap lines returned by read/search/filter tools. |
| `LOG_MCP_SEC_MAX_STREAM_BYTES` | Cap raw bytes read per stream when not set on query. |
| `LOG_MCP_SEC_MAX_LINE_BYTES` | Cap bytes per raw line. |
| `LOG_MCP_SEC_MAX_SCAN_LINES` | Upper bound for scan-style tools. |

## Run (stdio)

The host must spawn the binary with **stdin/stdout** connected (pipe or PTY per client).
Example for a local shell smoke check (not a full MCP session):

```bash
export LOG_MCP_ROOT=/var/log/myapp
./log-mcp
```

In practice, **Cursor**, **Claude Desktop**, or other MCP hosts supply the command and
env block in their config. Typical shape:

```json
{
  "mcpServers": {
    "log-mcp": {
      "command": "/absolute/path/to/log-mcp",
      "env": {
        "LOG_MCP_ROOT": "/var/log/myapp"
      }
    }
  }
}
```

Exact config keys depend on the host application; refer to its MCP documentation.


## Piped logs (shell)

MCP uses **stdin (fd 0)** for JSON-RPC, so you cannot attach raw application logs to
the same stdin as the server. Instead, open logs on **another fd** and set
**`LOG_MCP_PIPE_FD`** to that number (must be **≥ 3**). Tools then use **`stream_id`:
`pipe://`**.

Examples (bash):

```bash
# File on fd 3, MCP on stdin/stdout as usual
exec 3< /var/log/myapp/current
export LOG_MCP_PIPE_FD=3
./log-mcp
```

```bash
# Process substitution
exec 3< <(kubectl logs deploy/api -n prod --tail=500)
export LOG_MCP_PIPE_FD=3
./log-mcp
```

```bash
# One-shot: attach process substitution as fd 3 (bash, Linux /dev/fd)
LOG_MCP_PIPE_FD=3 ./log-mcp 3< <(cat access.log)
```

Do **not** pipe application logs into **stdin** while using MCP stdio — fd **0** must stay the MCP JSON-RPC stream from the host.

Each **`read_logs`** / scan call opens the fd again from the start until closed by
the reader; for one-shot pipes, a single tool read typically consumes the stream.

## Tool usage patterns

### Read recent lines

Call **`read_logs`** with `stream_id` set to the absolute file path (under
`LOG_MCP_ROOT`), a prefixed remote id (`k8s://…`, `docker://…`, `cf://…`), or **`pipe://`** when using `LOG_MCP_PIPE_FD`. Omit
filters and set `max_lines` for a bounded tail-oriented read.

### Search and filter

- **`search_logs`** — required `keyword`; optional `since`, `until`, `level` are **AND**ed with the keyword match (same line must satisfy all set predicates).
- **`filter_logs`** — supply **at least one** of `since`, `until`, `level`, `keyword`. **Every parameter you set is ANDed**: a line appears only if it matches the time window (if both ends set), level (if set), and keyword substring (if set). Omitted parameters are ignored. Use `filter_logs` for time-only or level-only slices; use `search_logs` when search text is the primary selector.

Times must be **RFC3339** (for example `2026-04-02T12:00:00Z`). String tool arguments must be valid JSON strings (e.g. `"level": "ERROR"`, `"keyword": "redis"`) when hosts serialize `tools/call` parameters.

**OR / DSL:** there is no built-in OR mode or query language; combine multiple tool calls client-side if needed. See [tools.md — `filter_logs`](tools.md#filter_logs).

### Incident workflow (deterministic)

1. **`summarize_errors`** on a stream → ranked `groups`.
2. **`detect_anomalies`** on the same or another stream → `level_spike_segments` + `repeated_errors` (if **`low_sample`** is true, spikes are omitted—stream too short or all lines in one segment).
3. **`incident_summary`** with `structured_summary` set to a merged JSON object from
   those tools (no raw `read_logs` payload).
4. Optionally **`explain_log_pattern`** with one `error_group` or `repeated_burst`.

### Trace / request correlation

1. **`group_by_field`** with `field: "trace_id"` (or `request_id`).
2. **`correlate_events`** with the chosen `field` and exact `value`.
3. **`extract_trace_flow`** for a compact ordered timeline with `trace_id` and optional
   `field` override.

## Release binaries

Tagged builds attach cross-platform archives; see [releases.md](releases.md).

## Related docs

- [troubleshooting.md](troubleshooting.md) — configuration and path issues.
- [development.md](development.md) — contributing and local test.
