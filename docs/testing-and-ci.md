# Testing and continuous integration

How **log-mcp** is verified in development and in GitHub Actions. For design and
sources, see [architecture.md](architecture.md); for tool JSON, see [tools.md](tools.md);
for contributor workflow, see [development.md](development.md).

## Go tests

From the repository root:

```bash
go test -count=1 ./...
go vet ./...
```

`internal/parse` embeds synthetic line fixtures under
[`internal/parse/testdata/fixtures/`](../internal/parse/testdata/fixtures/);
[`fixtures_test.go`](../internal/parse/fixtures_test.go) asserts that
[`AutoParser`](../internal/parse/text_parser.go) extracts expected substrings from
each file. See the fixtures [README](../internal/parse/testdata/fixtures/README.md)
for format references (platform logs, polyglot stacks, golden JSON schema).

Shared pointers for other packages live under [`test/fixtures/logs/`](../test/fixtures/logs/README.md).

### MCP stdio-style integration (`internal/server`)

[`integration_stdio_test.go`](../internal/server/integration_stdio_test.go) runs the
real **`github.com/modelcontextprotocol/go-sdk/mcp` JSON-RPC** session over
[`mcp.IOTransport`](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp#IOTransport)
(pipes), which uses the **same newline-delimited JSON framing** as
[`mcp.StdioTransport`](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp#StdioTransport).
The server is constructed with [`server.New`](../internal/server/server.go) and a
temporary **`LOG_MCP_ROOT`** so env-based wiring matches production. Tests include
**`filter_logs`** with arguments produced by **`encoding/json.Unmarshal`** from a full
JSON object (quoted strings and integer **`max_lines`**) to mirror host **`tools/call`**
payloads.

```bash
go test -count=1 ./internal/server/ -run Integration_stdio
```

## Dependency and security checks

CI runs **`govulncheck ./...`** against the Go version in [`go.mod`](../go.mod).
Locally:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

**`go mod tidy`** must leave [`go.mod`](../go.mod) and [`go.sum`](../go.sum)
unchanged; CI fails if a clean tidy would dirty the tree.

## Lint and static analysis

Pull requests run:

| Check | Role |
|-------|------|
| **no tracked AI/agent paths** | Fails if `.cursor/`, Copilot instructions, etc. are in the git index ([`check-no-ai-agent-paths.sh`](../.github/scripts/check-no-ai-agent-paths.sh)). |
| **go mod tidy** | Enforces clean module metadata. |
| **go vet** | Standard compile-time checks. |
| **golangci-lint** | Enabled linters in [`.golangci.yml`](../.golangci.yml); version pinned in [`.github/workflows/ci-reusable.yml`](../.github/workflows/ci-reusable.yml). |
| **gosec** | Security-oriented rules (`gosec ./...`). |
| **govulncheck** | Known vulnerability database for the module graph. |

Match **golangci-lint** to CI when running locally (see workflow file for the
current binary version).

## Documentation checks

Markdown under the repo (except `LICENSE`) is linted and link-checked (see
[docs-reusable workflow](../.github/workflows/docs-reusable.yml)). Prefer stable
external URLs in new links.

## Workflows overview

| Workflow | When | Notes |
|----------|------|-------|
| [ci.yml](../.github/workflows/ci.yml) | Push/PR to configured branches | Invokes reusable Go jobs. |
| [pr-check.yml](../.github/workflows/pr-check.yml) | PRs to `main` / `develop` | Gates on CI + docs. |
| [docs-ci.yml](../.github/workflows/docs-ci.yml) | Docs-focused runs | Link + markdown lint. |
| [benchmark.yml](../.github/workflows/benchmark.yml) | Selected branches | Benchmark job. |
| [release.yml](../.github/workflows/release.yml) | Version tags `v*` | Cross-compile `cmd/log-mcp`. |

## Release builds

Tagged releases build `log-mcp` for multiple OS/arch pairs and stamp version via
`-ldflags` (see [release.yml](../.github/workflows/release.yml)).
