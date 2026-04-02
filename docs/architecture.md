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
| Log query helpers | `internal/logtool` | Shared filter + collect over the pipeline (time, level, keyword). |
| Error summarization | `internal/errsummary` | Deterministic ERROR-level grouping, frequency ranking, per-service counts (no LLM). |
| Log correlation | `internal/logcorr` | Field-based grouping and timelines (`trace_id`, `request_id`, custom JSON keys). |
| Security | `internal/security` | CLI argv allowlists, read-only file checks, mandatory redaction entrypoint, env-driven I/O caps. |
| Build metadata | `internal/buildinfo` | Static name/version for MCP `Implementation` and `version_info`. |

## MCP layer (stdio)

The server uses the official Go SDK (`github.com/modelcontextprotocol/go-sdk/mcp`) with `StdioTransport`: **stdout** carries MCP messages; **stderr** is reserved for diagnostics (once logging is added). Wire envelopes are JSON-RPC 2.0; see `internal/server` for typed request/response structs used in tests.

### Log MCP tools

[`New`](../internal/server/server.go) builds log sources from the environment and registers tools when at least one source is configured:

- **`LOG_MCP_ROOT`** — filesystem roots (same separator as the OS `PATH`); see [File-backed source](#file-backed-source).
- **`LOG_MCP_K8S_NAMESPACES`** — comma-separated **allowlist** of Kubernetes namespace names (DNS labels only). Enables [`K8sLogSource`](../internal/source/k8s.go): only **`kubectl get pods -n <ns>`** and **`kubectl logs <pod> -n <ns>`** are run, with fixed argument shapes (no shell). Optional **`LOG_MCP_K8S_KUBECTL`** overrides the kubectl binary. Stream IDs are **`k8s://namespace/pod`** or **`k8s://namespace/pod/container`**.
- **`LOG_MCP_DOCKER_CONTAINERS`** — comma-separated **allowlist** of Docker container names. Enables [`DockerLogSource`](../internal/source/docker.go): **`docker ps -a --format '{{.Names}}'`** to list (intersected with the allowlist) and **`docker logs <name>`** with optional **`--tail`** from `max_lines`. Optional **`LOG_MCP_DOCKER_DOCKER`** overrides the docker binary. Stream IDs are **`docker://container_name`**.
- **`LOG_MCP_CF_APPS`** — comma-separated **allowlist** of Cloud Foundry app names (lowercase DNS-like). Enables [`CFLogSource`](../internal/source/cf.go): **`cf apps`** to list (intersected with the allowlist) and **`cf logs <app> --recent`**. Optional **`LOG_MCP_CF_CLI`** overrides the **`cf`** binary. Stream IDs are **`cf://app_name`**.
- When more than one of the above is set, [`MultiLogSource`](../internal/source/multi.go) merges the corresponding streams and routes reads by stream ID prefix.

Tools apply to any configured backend the same way. When nothing is configured, log tools return an error.

- **`read_logs`** — sequential read with optional `max_lines` and optional filters (`since` / `until` RFC3339, `level`, `keyword`). With no filters, `max_lines` is passed to the source (filesystem tail or kubectl `--tail`).
- **`search_logs`** — requires `keyword`; optional time and level filters; caps matches with `max_matches`.
- **`filter_logs`** — requires at least one of `since`, `until`, `level`, or `keyword`; optional `max_lines`.
- **`summarize_errors`** — scans up to `max_scan_lines` (default 20000; upper bound `LOG_MCP_SEC_MAX_SCAN_LINES`), keeps ERROR/FATAL/CRITICAL-style levels, groups lines by a stable fingerprint ([`errsummary`](../internal/errsummary/summary.go): redaction + normalized message and error-related JSON fields), sorts groups by **count descending** then fingerprint ascending, and reports **by_service** using `service` / `app` / `logger` / … fields or the log file basename.
- **`group_by_field`** — scans a capped prefix of the stream, groups lines that contain a JSON field. Logical names **`trace_id`** and **`request_id`** resolve common aliases (`traceId`, `x-trace-id`, `req_id`, `x-request-id`, …); any other `field` is matched case-insensitively on keys. Lines without the field are skipped; groups are ranked by **count descending**.
- **`correlate_events`** — returns all lines in the scan window whose field value **exactly** matches `value`, ordered by timestamp then line number (for timelines along one trace or request).

Outputs are **redacted** via [`internal/pipeline`](../internal/pipeline/pipeline.go) using [`security.RedactLine`](../internal/security/redact.go) (canonical [`normalize.DefaultConfig`](../internal/normalize/config.go); not configurable per tool). For tests or custom deployments, [`NewWithConfig`](../internal/server/server.go) accepts an explicit [`LogSource`](../internal/source/source.go).

### Security limits (`LOG_MCP_SEC_*`)

Optional caps (positive integers; invalid or empty env falls back to defaults):

- **`LOG_MCP_SEC_MAX_OUTPUT_LINES`** — max lines returned per `read_logs` / `search_logs` / `filter_logs` response (default 10000).
- **`LOG_MCP_SEC_MAX_STREAM_BYTES`** — max raw bytes read per stream when `MaxBytes` is unset on the query (default 50 MiB); applied by [`logtool.EnrichQuery`](../internal/logtool/query.go) before pipeline runs.
- **`LOG_MCP_SEC_MAX_LINE_BYTES`** — max bytes per raw line before redaction (default 1 MiB); applied in file and subprocess line readers when unset on the query.
- **`LOG_MCP_SEC_MAX_SCAN_LINES`** — upper bound for `summarize_errors`, `group_by_field`, and `correlate_events` scans (default 100000).

### Command allowlist and read-only I/O

[`ExecKubectlRunner`](../internal/source/k8s_runner.go), [`ExecDockerRunner`](../internal/source/docker_runner.go), and [`ExecCFRunner`](../internal/source/cf_runner.go) call [`security.ValidateKubectlArgs`](../internal/security/cli.go) / `ValidateDockerArgs` / `ValidateCFArgs` before every invocation; argv must match the fixed shapes used by the log sources (no arbitrary flags). Subprocess stdin is detached ([`security.HardenExecCmd`](../internal/security/exec.go)). Filesystem reads use `O_RDONLY` and [`security.FileReadableUnderRoots`](../internal/security/path.go) (symlink resolution) so paths cannot escape configured roots through links.

## Log source abstraction

Log backends implement [`LogSource`](../internal/source/source.go):

- **`Name()`** — Stable label for this source instance (metrics, tools, debugging).
- **`List(ctx)`** — Returns the set of **log streams** currently exposed (`LogStream` carries an `ID` and display `Name`).
- **`Read(ctx, query)`** — Opens a stream selected by `Query.StreamID` and returns a **receive-only channel** of `LogEntry` values (line number + raw line text). Reading stops when the file ends, the context is cancelled, or optional `MaxLines` / `MaxBytes` limits are hit.

This interface is intentionally **line-oriented**: parsers consume each `LogEntry.Raw` line without changing how bytes are pulled from the backend.

### Kubernetes backend

Pod log streams are enumerated per allowlisted namespace and read through kubectl. Namespace and pod (and optional container) names are validated as DNS labels before any subprocess call. Pod names returned by `kubectl get pods` are filtered the same way so unexpected output lines are ignored. Cluster access and credentials follow the normal **`kubectl`** / **`KUBECONFIG`** environment of the MCP server process.

### Docker backend

Container log streams are enumerated with **`docker ps -a`** (name format) and intersected with the allowlist. Only **`docker logs`** is used for reads (no `--follow`). Container names are validated with a conservative pattern before subprocess calls.

### Cloud Foundry backend

App streams are enumerated by parsing the tabular output of **`cf apps`**; only allowlisted app names that appear in that table are exposed. Reads use **`cf logs <app> --recent`** (bounded snapshot; line limits still apply when streaming the captured output). App names are validated before subprocess calls. The MCP server uses the same **`cf`** authentication and target org/space as the process environment.

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

Cancellation is honored at the **source**: when the [`context.Context`] passed to [`Pipeline.Run`] is cancelled, producers such as [`FileLogSource`](../internal/source/file.go) or [`K8sLogSource`](../internal/source/k8s.go) stop sending and close their output channel; workers drain connected channels and exit so the result channel closes.

### File-backed source

[`FileLogSource`](../internal/source/file.go) is configured with one or more **roots**, each an absolute path after construction:

- If a root is a **file**, it appears as a single stream.
- If a root is a **directory**, each **immediate** child regular file is a stream (no recursive descent yet).

`Read` resolves `Query.StreamID` to an absolute path and allows the open only if the resolved path (after `filepath.EvalSymlinks` on the target) is a regular file under one of the configured roots ([`security.FileReadableUnderRoots`](../internal/security/path.go)), so symlink escapes and arbitrary path injection are rejected when tools validate against `List` results. Opens use read-only mode (`os.O_RDONLY`).

Streaming uses a large `bufio.Reader` buffer and `ReadString('\n')` so individual lines can exceed the default `bufio.Scanner` token limit—important for “wide” log lines.

## Future direction (not implemented)

- MCP tool to call `LogSource.List` (stream discovery) and stricter validation of `stream_id` against listed IDs.
- Additional sources (journald, object storage) and richer Kubernetes APIs behind the same `LogSource` interface.
- Optional tail/follow semantics, nested JSON log models, and richer schema-aware parsers.
