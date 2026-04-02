# Security policy

## Supported versions

Security fixes are applied on active development lines. The default integration
branch is **`develop`**; releases are tagged as **`v*`** when published.

| Line | Supported |
|------|-----------|
| `develop` (latest commit) | Yes |
| Latest tagged release | Yes |
| Older tags / unmaintained snapshots | Best effort only |

## Reporting a vulnerability

**Please do not** file a public GitHub issue for undisclosed security problems.

Preferred options:

1. **GitHub private vulnerability reporting** when it is enabled for this
   repository: use the **Security** tab, then **Report a vulnerability**, as
   described in
   [GitHub Docs: Privately reporting a security vulnerability](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing/privately-reporting-a-security-vulnerability).
2. If that workflow is unavailable, use a **private** channel to reach the
   maintainers (for example contact options on a GitHub profile) and include:
   - A short description of the issue and its impact
   - Steps to reproduce or a proof-of-concept
   - Affected component (e.g. MCP tool, file source, subprocess runner)

We aim to acknowledge reports within a few business days and to coordinate
disclosure once a fix is ready.

## Scope and expectations

**log-mcp** is designed for **read-only** log inspection with allowlists and
I/O limits; hosts should still run the binary with **least privilege** and
configure **`LOG_MCP_*`** variables deliberately. Reports about misconfiguration
when the server is intentionally given broad filesystem or cluster access may
be treated as documentation or hardening requests rather than vulnerabilities.

## License

The software is distributed under the **Apache License 2.0**; see [LICENSE](LICENSE).
