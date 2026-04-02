# Documentation

Deep dives for **log-mcp** live in this directory.

## Start here

| Document | Description |
|----------|-------------|
| [architecture.md](architecture.md) | MCP stdio model, **all tools** and env vars, log sources (file, Kubernetes, Docker, Cloud Foundry), security limits (`LOG_MCP_SEC_*`), parsing, normalization, streaming pipeline. |

## Repository guides (root)

These files also appear in the GitHub repository **Community** / health checklist when present at the repo root:

| File | Description |
|------|-------------|
| [README.md](../README.md) | Project overview, **current MCP server behavior**, tool catalog, build, CI pointers. |
| [CONTRIBUTING.md](../CONTRIBUTING.md) | Branching, build/test commands, PR expectations. |
| [CODE_OF_CONDUCT.md](../CODE_OF_CONDUCT.md) | Contributor Covenant code of conduct. |
| [SECURITY.md](../SECURITY.md) | How to report vulnerabilities responsibly. |
| [LICENSE](../LICENSE) | Apache License 2.0. |

For automated checks, Markdown in the whole tree (except `LICENSE`) is linted and link-checked in CI.
