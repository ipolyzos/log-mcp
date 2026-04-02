# Releases

How **log-mcp** versioning and binary artifacts are produced.

## Branch flow

- **`develop`** — day-to-day integration; default target for pull requests.
- **`main`** — release-ready line when the project uses a promotion model; align with
  maintainer practice (some teams ship from `develop` only).

Recommended promotion flow when using both branches:

1. Merge feature work into **`develop`** with green CI.
2. When stable, open **`develop` → `main`** and merge after review and green checks.
3. Tag **`main`** (or the agreed release commit) as described below.

## Versioning

- Prefer **SemVer** `vMAJOR.MINOR.PATCH` tags (e.g. `v0.2.0`).
- [`internal/buildinfo`](../internal/buildinfo/buildinfo.go) supplies default
  `Version` for MCP `Implementation` and the `version_info` tool; release binaries
  should override it with **`-ldflags`** (see below).

The repository maintains **[`CHANGELOG.md`](../CHANGELOG.md)** (Keep a Changelog style)
alongside GitHub Releases. For **user-visible** behavior or tool changes, update the
**`[Unreleased]`** section in the same PR when practical; when you tag a release, move
those notes under the new version heading and link the tag in the changelog footer
(see existing **`[0.1.0]`** entry).

## Tagging and artifacts

Releases are driven by [`.github/workflows/release.yml`](../.github/workflows/release.yml):

- **Trigger:** push of a tag matching **`v*`**.
- **Build matrix:** `linux`/`darwin` × `amd64`/`arm64`, plus `windows/amd64`.
- **Artifacts:** per-platform archive (`.tar.gz` or `.zip`) containing the `log-mcp`
  binary (or `log-mcp.exe` on Windows).

The workflow sets:

```text
-ldflags "-s -w -X github.com/ipolyzos/log-mcp/internal/buildinfo.Version=${VERSION}"
```

where `VERSION` is the tag name (for example `v0.2.0`).

### Example: tag a new release

**v0.1.0** is already tagged; use this pattern for the next SemVer (for example **v0.2.0**):

```bash
git checkout main   # or develop, per project policy
git pull --ff-only origin main
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

Update **[`CHANGELOG.md`](../CHANGELOG.md)** with a **`[0.2.0]`** section before or
immediately after tagging so the file matches the published release notes.

Verify the **Release** workflow on GitHub and downloaded binaries.

## Backports

For **critical** fixes (especially security), maintainers may:

1. Branch from the release line (`fix/backport-…`).
2. Apply the minimal patch; PR into **`main`** (or the stable branch).
3. Cherry-pick or merge back into **`develop`** so branches do not diverge.

## Related docs

- [CHANGELOG.md](../CHANGELOG.md) — human-readable history by version.
- [usage.md](usage.md) — running released or local binaries.
- [development.md](development.md) — local build and test.
