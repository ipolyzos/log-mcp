# Development

Guide for contributors working on **log-mcp**. Community process (CoC, security
reporting) stays in the repo root; this file focuses on **code and docs**.

## Branching and pull requests

- Branch from **`develop`** (integration branch). Name prefixes that match CI push
  filters are convenient: `feature/`, `fix/`, `chore/`, `ci/` (see
  [`.github/workflows/ci.yml`](../.github/workflows/ci.yml)).
- Open PRs targeting **`develop`** unless maintainers say otherwise.
- Keep commits focused; merge when **PR Check** is green.

## Local checks

Before pushing:

```bash
gofmt -w .
go test -count=1 ./...
go vet ./...
go mod tidy
git diff --exit-code go.mod go.sum
```

Optional but recommended (match CI versions where possible):

```bash
golangci-lint run ./...
go install github.com/securego/gosec/v2/cmd/gosec@v2.25.0
gosec ./...
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

CI also enforces **markdown** and **link** checks on `*.md`; see
[testing-and-ci.md](testing-and-ci.md) for the workflow list and **`golangci-lint`**
pin.

## Parser fixtures

[`internal/parse`](../internal/parse/) embeds synthetic lines under
[`internal/parse/testdata/fixtures/`](../internal/parse/testdata/fixtures/). When you
adjust parsing behavior, update or add fixtures and expectations in
[`fixtures_test.go`](../internal/parse/fixtures_test.go). See the fixtures
[README](../internal/parse/testdata/fixtures/README.md).

## Documentation updates (same PR as behavior changes)

When you change tools, env vars, security limits, or pipeline semantics, update:

| Change touches | Update |
|----------------|--------|
| Tool params / JSON results | [tools.md](tools.md) |
| Error shapes / transport expectations | [client-contract.md](client-contract.md) |
| End-to-end flow, sources, limits | [architecture.md](architecture.md) |
| How to run / configure hosts | [usage.md](usage.md) |
| CI, tidy gate, fuzzing N/A | [testing-and-ci.md](testing-and-ci.md) |
| Release tagging / artifacts | [releases.md](releases.md) |
| One-line overview | [README.md](../README.md) (keep short; link into `docs/`) |

Also skim [troubleshooting.md](troubleshooting.md) if user-facing failure modes change.

## See also

- [CONTRIBUTING.md](../CONTRIBUTING.md) — PR checklist and license.
- [testing-and-ci.md](testing-and-ci.md) — CI jobs and dependency checks.
