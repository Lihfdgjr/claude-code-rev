# Security Policy

## Supported Versions

This project is in early development. Only the latest tagged release of the Go tree (`go/`) receives security fixes. The TypeScript tree at the repository root is reverse-engineered from public source maps and does not receive security support — use at your own risk.

| Version | Supported          |
| ------- | ------------------ |
| Latest tag (`go/`) | ✓ |
| Older tags | ✗ |
| TypeScript tree (`/`) | ✗ — restoration only |

## Reporting a Vulnerability

**Do not open a public issue for security vulnerabilities.**

Use one of:

1. **GitHub Security Advisory** (preferred): https://github.com/Lihfdgjr/claude-code-rev/security/advisories/new
2. Open a normal issue **without details**, asking the maintainer to contact you privately.

Please include:
- Affected file(s) and version
- Reproduction steps (minimal repro preferred)
- Impact assessment (data exposure? RCE? privilege escalation?)
- Any suggested fix (optional)

We aim to acknowledge within 7 days and patch within 30 days for critical issues.

## Scope

In scope:
- The Go binary built from `go/`
- The CI workflow files at `.github/workflows/`
- Dependencies declared in `go/go.mod`

Out of scope:
- The TypeScript tree at the repository root (use of those files is at your own risk)
- Third-party MCP servers users add to their own configuration
- Vulnerabilities in user-installed plugins (`~/.claude/plugins/`)
- Issues only reproducible with non-default configuration that disables permission gates

## Disclosure

We follow coordinated disclosure. Once a fix is available we publish a GitHub Security Advisory crediting the reporter (unless they prefer anonymity). Please give us a reasonable window before public disclosure.
