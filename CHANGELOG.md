# Changelog

All notable changes to **log-mcp** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **Docs / MCP ergonomics:** Documented **`filter_logs` AND semantics** end-to-end
  ([tools.md](docs/tools.md), [usage.md](docs/usage.md), [client-contract.md](docs/client-contract.md),
  [troubleshooting.md](docs/troubleshooting.md)); clarified the MCP tool description and
  JSON-schema hints. **OR-combine mode** and a **query DSL** remain **out of scope**
  until there is concrete integrator demand (documented in `tools.md`).

### Planned

- Nothing else listed yet; see [GitHub issues](https://github.com/ipolyzos/log-mcp/issues).

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

[Unreleased]: https://github.com/ipolyzos/log-mcp/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ipolyzos/log-mcp/releases/tag/v0.1.0
