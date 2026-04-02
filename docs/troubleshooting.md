# Troubleshooting

Common issues when running or integrating **log-mcp**.

## `log access not configured` on log tools

**Symptom:** `read_logs`, `search_logs`, or analysis tools return an error mentioning
`LOG_MCP_ROOT` or log access not configured.

**Cause:** No [`LogSource`](../internal/source/source.go) was constructed at startup.
At least one backend env var must be set (see [architecture.md](architecture.md#log-source-abstraction)).

**Fix:**

- Set **`LOG_MCP_ROOT`** to an absolute file or directory path (use the OS path-list
  separator for multiple roots), **or**
- Enable Kubernetes / Docker / Cloud Foundry allowlists (`LOG_MCP_K8S_NAMESPACES`, etc.).

`health_check` and `version_info` work without a source; `incident_summary` and
`explain_log_pattern` only need structured JSON input.

## Wrong or rejected `stream_id`

**Symptom:** Errors opening a stream, path denied, or unknown stream.

**Checks:**

- For **files**, `stream_id` is the **absolute path** under a configured root (as
  returned conceptually by listing; there is no `list_streams` tool yet — see
  [architecture.md](architecture.md#future-direction-not-implemented)).
- Path must stay under **`LOG_MCP_ROOT`** after symlink resolution; escapes are rejected.
- For **k8s/docker/cf**, IDs use prefixes like `k8s://…`, `docker://…`, `cf://…`
  as documented in [architecture.md](architecture.md#log-mcp-tools).

## `filter_logs` empty results or host JSON errors

**Symptom:** No lines returned though you expect matches, or the MCP host reports
**invalid JSON** / **invalid params** when calling `filter_logs`.

**Checks:**

- **AND semantics:** every **set** filter must match the same line. For example
  `level: ERROR` and `keyword: redis` returns only ERROR lines that contain `redis`.
  See [tools.md — `filter_logs`](tools.md#filter_logs).
- **JSON quoting:** arguments are JSON. Use quoted strings:
  `"level": "ERROR"`, `"keyword": "redis"` — not bare `ERROR` or `redis` tokens.
- **At least one filter:** you must pass at least one of `since`, `until`, `level`,
  `keyword` (see [client-contract.md](client-contract.md#typical-error-message-patterns)).

## `invalid RFC3339 time`

**Symptom:** Tool error when using `since` / `until`.

**Fix:** Use full **RFC3339** strings (for example `2026-04-02T15:04:05Z` or with a
numeric offset). Partial dates are not accepted.

## `kubectl`, `docker`, or `cf` failures

**Symptom:** Timeouts, exit errors, or empty streams when using remote backends.

**Checks:**

- Binary on **`PATH`** (or override env: `LOG_MCP_K8S_KUBECTL`, `LOG_MCP_DOCKER_DOCKER`,
  `LOG_MCP_CF_CLI`).
- Credentials and context match the server process (`KUBECONFIG`, `cf target`, etc.).
- Allowlists include the namespace/container/app you expect.

Only **fixed argv shapes** are permitted; arbitrary flags are rejected for safety
([architecture.md](architecture.md#command-allowlist-and-read-only-io)).

## Truncated or empty results

**Symptom:** `truncated: true` or fewer lines than expected.

**Cause:** Caps from tool defaults, `max_lines` / `max_matches` / `max_scan_lines`, and
**`LOG_MCP_SEC_*`** ([architecture.md](architecture.md#security-limits-log_mcp_sec_)).

**Fix:** Raise limits within policy, or scan a narrower time window / smaller stream.

## MCP / stdout corruption

**Symptom:** Client desync, parse errors, or “invalid JSON” from the host.

**Cause:** Something wrote **non-MCP** data to **stdout** (debug prints, subprocess
inheritance).

**Fix:** Send all diagnostics to **stderr** only. Do not wrap the binary in scripts
that echo to stdout.

## CI passes locally but fails in GitHub Actions

**Checks:**

- Run **`go mod tidy`** and commit `go.mod` / `go.sum` (CI enforces a clean tidy).
- Match **Go** version from [`go.mod`](../go.mod).
- Match **golangci-lint** version from [`.github/workflows/ci-reusable.yml`](../.github/workflows/ci-reusable.yml).

See [testing-and-ci.md](testing-and-ci.md).

## LLM helper tools reject input

**Symptom:** `incident_summary` or `explain_log_pattern` validation errors.

**Fix:**

- **`incident_summary`:** Pass a **`structured_summary`** object shaped like merged
  **`summarize_errors`** / **`detect_anomalies`** output — not raw `read_logs.lines`.
- **`explain_log_pattern`:** Supply **exactly one** of **`error_group`** or
  **`repeated_burst`**, matching the shapes in [tools.md](tools.md).

## Related docs

- [client-contract.md](client-contract.md) — how errors surface over MCP.
- [usage.md](usage.md) — configuration and running the server.
