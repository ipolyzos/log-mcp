# Architecture

This document describes how **log-mcp** is structured today and how log data flows through the system. For repository layout and build instructions, see the root [README.md](../README.md).

## Goals

- **Read-only** access to log data: no writes, deletes, or shell execution on behalf of tools.
- **Streaming** reads so large files do not need to be loaded entirely into memory.
- **Testable** boundaries: transports and sources can be exercised without a full MCP client.

## Major components

| Area | Location | Role |
|------|----------|------|
| MCP server | `cmd/log-mcp`, `internal/server` | stdio JSON-RPC session, tool registration, process lifecycle. |
| Tool registry | `internal/tools` | Ordered registration of MCP tools onto `mcp.Server`. |
| Log sources | `internal/source` | Pluggable backends that list streams and read them as lines. |
| Log parsing | `internal/parse` | JSON and text line parsers; shared `ParsedLine` (timestamp, level, message, fields). |
| Normalization | `internal/normalize` | Deterministic masking (UUID, IP, tokens) and normalization (timestamps, dynamic IDs). |
| Processing pipeline | `internal/pipeline` | Bounded-channel stream: source → parse → normalize → analyze (no full-buffering of the log). |
| Build metadata | `internal/buildinfo` | Static name/version for MCP `Implementation` and `version_info`. |

## MCP layer (stdio)

The server uses the official Go SDK (`github.com/modelcontextprotocol/go-sdk/mcp`) with `StdioTransport`: **stdout** carries MCP messages; **stderr** is reserved for diagnostics (once logging is added). Wire envelopes are JSON-RPC 2.0; see `internal/server` for typed request/response structs used in tests.

## Log source abstraction

Log backends implement [`LogSource`](../internal/source/source.go):

- **`Name()`** — Stable label for this source instance (metrics, tools, debugging).
- **`List(ctx)`** — Returns the set of **log streams** currently exposed (`LogStream` carries an `ID` and display `Name`).
- **`Read(ctx, query)`** — Opens a stream selected by `Query.StreamID` and returns a **receive-only channel** of `LogEntry` values (line number + raw line text). Reading stops when the file ends, the context is cancelled, or optional `MaxLines` / `MaxBytes` limits are hit.

This interface is intentionally **line-oriented**: parsers consume each `LogEntry.Raw` line without changing how bytes are pulled from disk.

## Log parsing

Package [`internal/parse`](../internal/parse/parsed.go) turns a single line into a `ParsedLine`:

- **`Timestamp`** — optional `time.Time` when a value could be resolved (RFC3339 strings, unix seconds/millis in JSON, RFC3339-ish prefix in text).
- **`Level`** — normalized uppercase level (`INFO`, `WARN`, `ERROR`, …).
- **`Message`** — primary human-readable payload when detected; otherwise the trimmed line.
- **`Fields`** — remaining key/value pairs: in JSON, keys not consumed as timestamp/level/message; in text, `key=value` / `key="value"` segments.

**`JSONParser`** accepts one JSON **object** per line and maps common keys (`@timestamp`, `timestamp`, `ts`, `level`, `msg`, …). Values are flattened to strings in `Fields` (non-scalars are JSON-encoded).

**`TextParser`** uses lightweight regex heuristics: optional RFC3339 prefix, `[LEVEL]` or bare level token, then `key=value` pairs stripped from the message body.

**`AutoParser`** tries JSON when the trimmed line starts with `{`; invalid JSON falls back to `TextParser` so mixed log files still yield a useful `ParsedLine`.

## Log normalization

Package [`internal/normalize`](../internal/normalize/normalize.go) implements [`Apply`](../internal/normalize/normalize.go) with a [`Config`](../internal/normalize/config.go) (see [`DefaultConfig`](../internal/normalize/config.go)): a **fixed-order**, **deterministic** pipeline so the same input string always yields the same redacted output (stable for tests and clustering).

**Masking (placeholders):** UUIDs (`<uuid>`), IPv6 then IPv4 (`<ipv6>`, `<ipv4>`), and tokens including JWT-shaped segments, `Bearer …`, `Authorization: Bearer …`, common `api_key` / `token` / `password` assignments, `sk-*`, and `ghp_*` / `ghs_*` style prefixes (`<token>`).

**Normalization:** RFC3339-like timestamps and 10-digit unix seconds (UTC, years 2000–2100) are rewritten to a canonical UTC string; long decimal runs (15–22 digits) become `<num_id>` and 16–40 character hex runs `<hex_id>` (after UUID masking so dashed UUIDs are not double-handled).

## Streaming processing pipeline

Package [`internal/pipeline`](../internal/pipeline/pipeline.go) composes existing pieces into a **line-at-a-time** pipeline:

1. **Source** — [`LogSource.Read`](../internal/source/source.go) yields `LogEntry` values on a channel (same contract as today).
2. **Parser** — each `Raw` line is parsed with a [`LineParser`](../internal/pipeline/pipeline.go) (default [`parse.AutoParser`](../internal/parse/text_parser.go)).
3. **Normalizer** — [`normalize.Apply`](../internal/normalize/normalize.go) runs on the **original** raw line; the structured view for downstream use is produced by **parsing again** on the redacted string so `Parsed` fields match the masked bytes.
4. **Analyzer** — optional [`Analyzer`](../internal/pipeline/pipeline.go) hook receives [`NormalizedEntry`](../internal/pipeline/pipeline.go) per line (metrics, routing, etc.).

Stages communicate through **Go channels** whose capacity is set by [`Pipeline.ChannelBuffer`](../internal/pipeline/pipeline.go) (default 32). Only a bounded number of in-flight `LogEntry` / intermediate structs exist at once; the implementation does **not** read the whole file into a slice before processing.

Cancellation is honored at the **source**: when the [`context.Context`] passed to [`Pipeline.Run`] is cancelled, producers such as [`FileLogSource`](../internal/source/file.go) stop sending and close their output channel; workers drain connected channels and exit so the result channel closes.

### File-backed source

[`FileLogSource`](../internal/source/file.go) is configured with one or more **roots**, each an absolute path after construction:

- If a root is a **file**, it appears as a single stream.
- If a root is a **directory**, each **immediate** child regular file is a stream (no recursive descent yet).

`Read` resolves `Query.StreamID` to an absolute path and allows the open only if that path is contained under one of the configured roots (via `filepath.Rel`), so arbitrary path injection from tool arguments can be rejected when tools validate against `List` results.

Streaming uses a large `bufio.Reader` buffer and `ReadString('\n')` so individual lines can exceed the default `bufio.Scanner` token limit—important for “wide” log lines.

## Future direction (not implemented)

- MCP tools that call `LogSource.List` / `Read` and the streaming pipeline with redaction and rate limits.
- Additional sources (journald, object storage, Kubernetes API) behind the same `LogSource` interface.
- Optional tail/follow semantics, nested JSON log models, and richer schema-aware parsers.
