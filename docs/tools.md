# MCP tools reference

Tools exposed by **log-mcp** via the MCP **`tools/call`** flow (Go SDK registers
names and JSON Schema from struct tags). All log-line payloads are **redacted**
([`security.RedactLine`](../internal/security/redact.go)) before leaving the pipeline.

**Registration order:** `health_check` → `version_info` → log tools → LLM helpers
([`internal/server/server.go`](../internal/server/server.go)).

For transport and error behavior, see [client-contract.md](client-contract.md).
For environment variables that enable backends, see [architecture.md](architecture.md#log-mcp-tools).

---

## `health_check`

Liveness-style probe.

| Direction | Field | Type | Required | Notes |
|-----------|-------|------|----------|-------|
| Params | _(none)_ | | | Empty object. |
| Result | `status` | string | yes | e.g. `"ok"`. |

---

## `version_info`

Build metadata from [`internal/buildinfo`](../internal/buildinfo/buildinfo.go).

| Direction | Field | Type | Required |
|-----------|-------|------|----------|
| Result | `name` | string | yes |
| Result | `version` | string | yes |

Release builds should stamp `version` via `-ldflags` (see [releases.md](releases.md)).

---

## `read_logs`

Sequential read with optional filters. If **no** time/level/keyword/`keywords_any` filters are set,
reads up to **`max_lines`** without scanning the entire file (tail-style where the
source supports it).

| Direction | Field | Type | Required | Notes |
|-----------|-------|------|----------|-------|
| Params | `stream_id` | string | yes | Stream identifier from the configured source. |
| Params | `max_lines` | int64 | no | Default **500**; capped by `LOG_MCP_SEC_MAX_OUTPUT_LINES`. |
| Params | `since` | string | no | RFC3339 inclusive lower bound. |
| Params | `until` | string | no | RFC3339 inclusive upper bound. |
| Params | `level` | string | no | e.g. `INFO`, `WARN`, `ERROR`. |
| Params | `keyword` | string | no | Case-insensitive substring on **redacted** line. Mutually exclusive with `keywords_any` in one call. |
| Params | `keywords_any` | string[] | no | OR list: line must contain **at least one** term (case-insensitive). Max **32** terms, **256** bytes each after trim. Mutually exclusive with `keyword`. |
| Result | `stream_id` | string | yes | Echo. |
| Result | `lines` | array | yes | Objects: `line_number`, `timestamp?`, `level?`, `message?`, `text`. |
| Result | `truncated` | bool | yes | True if more lines existed beyond caps. |
| Result | `result_hint` | string | no | When `lines` is empty and not `truncated`: **`no_lines_in_stream`** if no filters were used; **`no_matching_lines`** if filters excluded every line. Omitted when lines exist or when truncated with zero lines. |

---

## `search_logs`

Keyword search over redacted lines.

| Direction | Field | Type | Required | Notes |
|-----------|-------|------|----------|-------|
| Params | `stream_id` | string | yes | |
| Params | `keyword` | string | yes | Non-empty substring. |
| Params | `max_matches` | int64 | no | Default **500**; capped by security max output lines. |
| Params | `since`, `until`, `level` | string | no | Same semantics as `read_logs`. |
| Result | | | | Same shape as `read_logs` result (`result_hint` **`no_matching_lines`** when zero hits and not truncated). |

---

## `filter_logs`

Return lines where **every** filter you pass is satisfied (**boolean AND**).

### Semantics

| Rule | Detail |
|------|--------|
| **Minimum** | At least **one** of `since`, `until`, `level`, `keyword`, `keywords_any` must be set (non-empty after trim / non-empty list). Otherwise the tool errors. |
| **AND** | If you set multiple predicates, a line is returned only if it matches **all** of them. Omitted parameters are not applied. |
| **Time** | `since` / `until` are **inclusive** RFC3339 bounds on the parsed timestamp when present; both may be set → line must fall in `[since, until]`. |
| **Level** | Exact match on the normalized level string the pipeline extracted (e.g. `ERROR`), when that field is set. |
| **Keyword** | Case-insensitive substring on the **redacted** full line text (`text`). Do not set together with `keywords_any`. |
| **keywords_any** | OR across terms: the **redacted** line must contain **at least one** entry (case-insensitive). Capped at **32** terms, **256** bytes per term. Mutually exclusive with `keyword` in the same request. |

### Contrast with `search_logs`

- **`search_logs`** requires **`keyword`** and optionally ANDs `since` / `until` / `level`.
- **`filter_logs`** does **not** require `keyword`; use **`keywords_any`** when you need **OR** substrings (with time/level still ANDed). Use it for **level-only** or **time-only** slices, or **level + keyword** without treating keyword as the primary search axis.

### Examples (arguments only)

```json
{
  "stream_id": "/var/log/app.log",
  "level": "ERROR",
  "max_lines": 100
}
```

```json
{
  "stream_id": "/var/log/app.log",
  "since": "2026-04-02T00:00:00Z",
  "until": "2026-04-02T23:59:59Z",
  "keyword": "timeout",
  "max_lines": 50
}
```

Second example: lines must be **inside the window** **and** contain `timeout` (and respect `max_lines`).

```json
{
  "stream_id": "/var/log/app.log",
  "level": "ERROR",
  "keywords_any": ["timeout", "connection refused", "503"],
  "max_lines": 50
}
```

Third example: **ERROR** lines that contain **any** of the listed substrings.

### OR on substrings

Use **`keywords_any`** on **`filter_logs`** (and optionally **`read_logs`**) for **OR** matching on the redacted line. **`search_logs`** remains single-`keyword` only. There is still **no** general query DSL; combine with multiple tool calls when you need richer logic.

| Direction | Field | Type | Required | Notes |
|-----------|-------|------|----------|-------|
| Params | `stream_id` | string | yes | |
| Params | `max_lines` | int64 | no | Default **500**. |
| Params | `since`, `until`, `level`, `keyword`, `keywords_any` | string / string[] | no | At least **one** predicate must be non-empty; **all set fields** are **AND**ed; `keyword` xor `keywords_any` for text. |
| Result | | | | Same shape as `read_logs` result (`result_hint` **`no_matching_lines`** when zero hits and not truncated). |

---

## `summarize_errors`

Deterministic ERROR/FATAL/CRITICAL grouping (no LLM). Scans from the start of the
stream up to **`max_scan_lines`** (default **20000**, upper bound
**`LOG_MCP_SEC_MAX_SCAN_LINES`**).

| Direction | Field | Type | Required | Notes |
|-----------|-------|------|----------|-------|
| Params | `stream_id` | string | yes | |
| Params | `max_scan_lines` | int64 | no | Scan cap; clamped to security limit. |
| Result | `stream_id` | string | yes | |
| Result | `scanned_lines` | int64 | yes | |
| Result | `error_line_count` | int64 | yes | |
| Result | `truncated` | bool | yes | |
| Result | `groups` | array | yes | Each: `rank`, `count`, `fingerprint`, `pattern`, `by_service` (`{service,count}[]`). |

---

## `group_by_field`

Groups lines that contain a JSON field. Logical names **`trace_id`** and
**`request_id`** include common key aliases.

| Direction | Field | Type | Required | Notes |
|-----------|-------|------|----------|-------|
| Params | `stream_id` | string | yes | |
| Params | `field` | string | yes | `trace_id`, `request_id`, or custom JSON key. |
| Params | `max_scan_lines` | int64 | no | Default **20000**. |
| Params | `max_groups` | int64 | no | Default **50** (hard cap **500**). |
| Result | `stream_id`, `field` | string | yes | |
| Result | `scanned_lines` | int64 | yes | |
| Result | `truncated` | bool | yes | |
| Result | `groups` | array | yes | `value`, `count`, `first_line`, `last_line`, `line_numbers?`. |

---

## `correlate_events`

All lines whose field value **exactly** matches **`value`**, ordered by timestamp then
line number.

| Direction | Field | Type | Required | Notes |
|-----------|-------|------|----------|-------|
| Params | `stream_id` | string | yes | |
| Params | `field` | string | yes | Same logical names as `group_by_field`. |
| Params | `value` | string | yes | Exact match. |
| Params | `max_scan_lines` | int64 | no | Default **20000**. |
| Params | `max_events` | int64 | no | Default **500**. |
| Result | `stream_id`, `field`, `value` | string | yes | |
| Result | `scanned_lines` | int64 | yes | |
| Result | `truncated` | bool | yes | |
| Result | `events` | array | yes | Same line object shape as `read_logs.lines`. |

---

## `detect_anomalies`

Segment-based error-rate spikes and repeated identical ERROR lines (see
[`internal/loganalysis`](../internal/loganalysis/anomaly.go)).

| Direction | Field | Type | Required | Notes |
|-----------|-------|------|----------|-------|
| Params | `stream_id` | string | yes | |
| Params | `max_scan_lines` | int64 | no | Default **20000**; capped by `LOG_MCP_SEC_MAX_SCAN_LINES`. |
| Params | `segment_count` | int64 | no | ≥ **2**; default **15**. |
| Params | `z_score_threshold` | float64 | no | Default **2**. |
| Params | `min_repeat_errors` | int64 | no | Default **6**. |
| Params | `include_warn_bursts` | bool | no | When **true**, also report highly repeated identical **WARN**/**WARNING** lines (same threshold as errors). |
| Result | `stream_id` | string | yes | |
| Result | `scanned_lines` | int64 | yes | |
| Result | `truncated` | bool | yes | |
| Result | `low_sample` | bool | yes | **true** when `scanned_lines` &lt; `segment_count` or fewer than two segments received lines; **level_spike_segments** is then empty (z-scores need at least two populated segments). |
| Result | `segment_count` | int | yes | Number of line-based segments used for bucketing (default **15**). |
| Result | `active_segment_count` | int | yes | Segments that received at least one line. |
| Result | `level_spike_segments` | array | yes | `segment_index`, `line_start`, `line_end`, `error_count`, `mean_errors`, `std_dev`, `z_score`. |
| Result | `repeated_errors` | array | yes | `fingerprint`, `sample_message`, `count`, `first_line`, `last_line`. |
| Result | `repeated_warn_bursts` | array | no | Same shape as `repeated_errors`; present only when `include_warn_bursts` was true and qualifying WARN lines exist. |

---

## `extract_trace_flow`

Ordered steps for one trace (or correlated id); field defaults to logical **`trace_id`**.

| Direction | Field | Type | Required | Notes |
|-----------|-------|------|----------|-------|
| Params | `stream_id` | string | yes | |
| Params | `trace_id` | string | yes | Value as it appears in logs. |
| Params | `field` | string | no | Logical field name; default `trace_id`. |
| Params | `max_scan_lines` | int64 | no | Passed through to correlation scan. |
| Params | `max_steps` | int64 | no | Caps number of steps (events). |
| Result | `stream_id`, `field`, `trace_id` | string | yes | |
| Result | `scanned_lines` | int64 | yes | |
| Result | `truncated` | bool | yes | |
| Result | `steps` | array | yes | `step`, `line_number`, `timestamp?`, `service?`, `span_id?`, `level?`, `message?`, `text`. |

---

## `incident_summary`

**Does not read logs.** Consumes structured JSON only.

| Direction | Field | Type | Required | Notes |
|-----------|-------|------|----------|-------|
| Params | `structured_summary` | object | yes | Merge of **`summarize_errors`** and/or **`detect_anomalies`** top-level objects. |
| Result | `headline` | string | yes | |
| Result | `bullets` | string[] | yes | When **`detect_anomalies`** included **`low_sample`: true**, the first bullet notes that segment z-scores were omitted. |
| Result | `for_llm_context` | string | yes | |
| Result | `summary_kind_hint` | string | yes | |

---

## `explain_log_pattern`

**Does not read logs.** Exactly **one** of the following must be set.

| Direction | Field | Type | Required | Notes |
|-----------|-------|------|----------|-------|
| Params | `error_group` | object | one-of | Shape: `rank`, `count`, `fingerprint`, `pattern` (one `summarize_errors.groups[]` element). |
| Params | `repeated_burst` | object | one-of | Shape: `fingerprint`, `sample_message`, `count`, `first_line`, `last_line` (one `detect_anomalies.repeated_errors[]` element). |
| Result | `pattern_kind` | string | yes | |
| Result | `overview` | string | yes | |
| Result | `interpretation` | string[] | yes | |
| Result | `llm_prompt_hints` | string[] | yes | |

---

## Related docs

- [architecture.md](architecture.md) — pipeline, parsing, security limits.
- [usage.md](usage.md) — examples and configuration.
- [troubleshooting.md](troubleshooting.md) — common errors.
