# Changelog

All notable changes to **log-mcp** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **CRI stdout HTTP 5xx detection** ([internal/parse/text_parser.go](internal/parse/text_parser.go)): infer ERROR for access-log lines without an `HTTP/x.y` token (for example `GET /path 502`) and for structured `status=`, `http_status=`, `status_code=`, `response_status=`, and `resp_status=` fields in the 500–599 range.

### Planned

- Nothing else listed yet; see [GitHub issues](https://github.com/ipolyzos/log-mcp/issues).

---

## [0.1.2] — 2026-04-02

Operator-focused parsing and anomaly UX: Kubernetes **CRI** severity heuristics, **`detect_anomalies`** **`low_sample`**, and **`incident_summary`** awareness; MCP stdio integration tests.

### Added

- **MCP stdio-style integration tests** in [internal/server/integration_stdio_test.go](internal/server/integration_stdio_test.go): JSON-RPC over piped **[`mcp.IOTransport`](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp#IOTransport)** (same framing as stdio), temp **`LOG_MCP_ROOT`**, **`filter_logs`** with full JSON arguments (`json.Unmarshal`), plus **`read_logs`**, **`health_check`**, and **`version_info`**.
- **Kubernetes CRI plain-text parsing** in [internal/parse/text_parser.go](internal/parse/text_parser.go): recognize `stdout` / `stderr` with CRI tag **`F`** or **`P`** after an optional RFC3339 prefix; emit **`cri_stream`**; infer **ERROR** on **stderr** when the message matches operator-oriented failure needles (connection errors, timeouts, panic/fatal, permission/denied, etc.) so **`summarize_errors`** and **`detect_anomalies`** count those lines like other error-level logs.

### Changed

- **`detect_anomalies`** ([internal/loganalysis/anomaly.go](internal/loganalysis/anomaly.go), [internal/tools/analysis_tools.go](internal/tools/analysis_tools.go)): result includes **`low_sample`** when **`scanned_lines`** is less than **`segment_count`** or fewer than two segments received lines. **Level spike z-scores** are computed only across **segments that actually contain lines**; with fewer than two such segments, **`level_spike_segments`** is empty so tiny streams are not given misleading geographic spikes.
- **Kubernetes CRI stdout + HTTP 5xx** in [internal/parse/text_parser.go](internal/parse/text_parser.go): after stripping the CRI prefix, infer **ERROR** when the message matches `HTTP/x.y 5xx` (access-log style) so probe/upstream failures on stdout surface in **`summarize_errors`** / **`detect_anomalies`** like stderr-derived errors. Fixture `01_kubernetes_cri.log` includes an example **503** line.
- **`incident_summary`** ([internal/llmtools/summary.go](internal/llmtools/summary.go)): when merged **`detect_anomalies`** JSON includes **`low_sample`: true**, the brief adds a leading bullet and **for_llm_context** guidance so operators do not treat omitted segment spikes as geographic evidence.
- **Tests:** [internal/loganalysis/anomaly_test.go](internal/loganalysis/anomaly_test.go) covers **`detect_anomalies`** on Kubernetes **CRI-shaped** lines (stderr failure + stdout **5xx**) with **`low_sample`** and empty **`level_spike_segments`**.

### Notes

- Default **`internal/buildinfo.Version`** for non-`-ldflags` builds is **0.1.2**; release archives still stamp the tag via [`.github/workflows/release.yml`](.github/workflows/release.yml).

---

## [0.1.1] — 2026-04-02

Documentation and developer-ergonomics release (no change to log tool behavior or MCP JSON shapes).

### Changed

- Document **`filter_logs`** **AND** semantics end-to-end: [docs/tools.md](docs/tools.md), [docs/usage.md](docs/usage.md), [docs/client-contract.md](docs/client-contract.md), [docs/troubleshooting.md](docs/troubleshooting.md), [docs/architecture.md](docs/architecture.md), and the **`filter_logs`** MCP tool description / `jsonschema` hints in [internal/tools/log_tools.go](internal/tools/log_tools.go). Explicitly defer **OR** predicates and a **query DSL** until there is integrator demand.
- Align **release** and **contributor** docs with the tracked changelog: [docs/releases.md](docs/releases.md), [docs/README.md](docs/README.md), [README.md](README.md), [docs/development.md](docs/development.md).
- **Markdownlint:** set **`MD024.siblings_only: true`** in [`.markdownlint.yaml`](.markdownlint.yaml) so Keep a Changelog section headings validate.

### Notes

- Default **`internal/buildinfo.Version`** for non-`-ldflags` builds is **0.1.1**; release archives still stamp the tag via [`.github/workflows/release.yml`](.github/workflows/release.yml).

---

## [0.1.0] — 2026-04-02

First stable release line: Model Context Protocol server for **read-only**, **redacted** log access, parsing, summarization, correlation, and deterministic analysis. Default binary version (`internal/buildinfo`) is `0.1.0`; release artifacts use the `v0.1.0` tag via `-ldflags` (see [docs/releases.md](docs/releases.md)).

### Added

#### Core platform

- **MCP server** (`cmd/log-mcp`) on stdio with a registered tool surface and health/version probes.
- **Log sources**: file-backed streams under configurable roots (`LOG_MCP_ROOT`), plus optional **Kubernetes**, **Docker**, and **Cloud Foundry** backends (multi-source routing by stream id prefix).
- **Security layer**: path allowlists, scan/output limits, symlink-safe opens, and **redaction** of sensitive patterns before tools return data (`LOG_MCP_SEC_*` env tuning).
- **Streaming pipeline**: line-oriented processing with JSON and text parsing, normalization, and masking.

#### MCP tools — retrieval

- **`read_logs`** — sequential read with optional time window, level, keyword, and line cap.
- **`search_logs`** — keyword search on redacted lines with optional filters.
- **`filter_logs`** — lines matching **all** of the optional time / level / keyword predicates (AND semantics).

#### MCP tools — errors & analysis

- **`summarize_errors`** — deterministic grouping of ERROR/FATAL/CRITICAL lines by fingerprint, ranked counts, per-service aggregates.
- **`detect_anomalies`** — segment-wise error-rate z-scores and detection of highly repeated identical ERROR lines (no ML).
- **`extract_trace_flow`** — ordered timeline for a single `trace_id` (or `request_id` alias), compact steps.

#### MCP tools — correlation

- **`group_by_field`** — bucket lines by `trace_id`, `request_id`, or a JSON field (with common aliases).
- **`correlate_events`** — all lines sharing one field value, time-ordered.

#### MCP tools — LLM-oriented (structured input only)

- **`incident_summary`** — deterministic briefing from merged `summarize_errors` / `detect_anomalies` JSON (no raw log read).
- **`explain_log_pattern`** — interpretation hints for one `summarize_errors` group or one `detect_anomalies` repeated burst.

#### MCP tools — meta

- **`health_check`** — liveness.
- **`version_info`** — server name and version string.

#### Parsing & fixtures

- Log **parser** for structured JSON logs and plain text, with normalization for downstream tools.
- Large **parse fixture set** under `internal/parse/testdata/fixtures/` covering syslog-style and polyglot app logs (containers, proxies, databases, cloud providers, Java / Python / JavaScript / Rust / C / C++ / C#, golden polyglot stack, etc.).

#### Documentation & community

- **Docs hub** (`docs/`): tools, usage, client contract, development, testing/CI, releases, architecture.
- **Root README** aligned with the hub; **SECURITY**, **CONTRIBUTING**, **CODE_OF_CONDUCT**, **LICENSE**.

#### CI & automation

- **CI**: build, lint (`golangci-lint` v2), test, **go mod tidy** check, docs link validation (**lychee**), benchmarks workflow, **security** scanning workflow.
- **Release workflow**: on push of tags `v*`, cross-compile archives (Linux/Darwin/Windows × amd64/arm64 where applicable) with `buildinfo.Version` set from the tag; **GitHub Release** with generated notes and uploaded artifacts.
- **Repository hygiene**: expanded `.gitignore` and CI guard to avoid committing Cursor/Copilot/agent local config paths.

### Changed

- **Go toolchain** raised to **1.26.1** (module `go` directive).
- **MCP SDK**: `github.com/modelcontextprotocol/go-sdk` at **v1.4.1**.

### Fixed

- Parser / pipeline edge cases addressed in early PRs (e.g. token mask init, file close errcheck in stream goroutine, fixture doc links for lychee).

### Security

- Enforced configurable limits and redaction on all log paths exposed through MCP; see [SECURITY.md](SECURITY.md) and [docs/client-contract.md](docs/client-contract.md).

---

[Unreleased]: https://github.com/ipolyzos/log-mcp/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/ipolyzos/log-mcp/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/ipolyzos/log-mcp/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/ipolyzos/log-mcp/releases/tag/v0.1.0
