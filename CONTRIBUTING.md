# Contributing to log-mcp

Thank you for your interest in improving **log-mcp**. This document explains how
we work day to day and what we expect from contributions.

## Code of conduct

Participation is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By
engaging with the project you agree to uphold it.

## Security

Do **not** open public issues for unfixed security vulnerabilities. Follow
[SECURITY.md](SECURITY.md) instead.

## Getting started

1. **Fork** the repository and **clone** your fork.
2. Create a branch from **`develop`** (the integration branch used in CI).
3. Make focused commits with clear messages.
4. Open a **pull request** targeting **`develop`** (unless maintainers ask
   otherwise).

Branch names that match CI push filters are convenient for running the full
workflow early: `feature/…`, `fix/…`, `chore/…`, `ci/…`, etc. (see
[`.github/workflows/ci.yml`](.github/workflows/ci.yml)).

## Development requirements

- **Go** version compatible with [`go.mod`](go.mod) (see the `go` directive;
  CI installs that exact toolchain via `go-version-file`).
- Optional but recommended: **`golangci-lint`** at the version pinned in
  [`.github/workflows/ci-reusable.yml`](.github/workflows/ci-reusable.yml).

## Build and test

From the repository root:

```bash
go build -o log-mcp ./cmd/log-mcp
go test -count=1 ./...
go vet ./...
```

If you have `golangci-lint` installed:

```bash
golangci-lint run ./...
```

Security-related checks in CI also run **`gosec`** and **`govulncheck`**; running
them locally before pushing reduces iteration time:

```bash
go install github.com/securego/gosec/v2/cmd/gosec@v2.25.0
gosec ./...
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

## Documentation and links

Markdown is linted in CI (`markdownlint-cli2`). **Relative links** in new or
updated docs should resolve within the repository. External URLs should be
stable (official docs, standards).

Authoritative technical detail belongs in [`docs/`](docs/) (see
[`docs/README.md`](docs/README.md) and [`docs/development.md`](docs/development.md));
keep the root [`README.md`](README.md) as a short map into that folder when you
change user-visible behavior.

Before pushing dependency changes, run **`go mod tidy`** and ensure
`go.mod` / `go.sum` are committed (CI enforces a clean tidy).

## Pull request checklist

- [ ] `go test ./...` and `go vet ./...` pass locally.
- [ ] Changes are scoped to the problem you are solving (avoid unrelated
      refactors).
- [ ] User-visible behavior or configuration changes are reflected in the right
      `docs/*.md` files (see [`docs/development.md`](docs/development.md)) and the
      root `README.md` only if a one-line pointer or summary is needed.

## License

By contributing, you agree that your contributions will be licensed under the
same terms as the project: **Apache License 2.0** (see [LICENSE](LICENSE)).
