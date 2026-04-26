# Contributing

Thanks for your interest. A few ground rules.

## Scope of this repository

Two independent trees live here:

| Tree | Purpose | How it was built |
|------|---------|------------------|
| `/` (TypeScript) | Restored Claude Code source | Reconstructed from public source maps with shim/fallback for unrecoverable modules |
| `/go/` | Independent Go reimplementation | Written from scratch; **must not** copy code, structure, or wording from the TS tree |

Keep contributions to one tree per PR. Do not mix.

## License

By contributing you agree to release your changes under
[PolyForm Noncommercial 1.0.0](LICENSE). Commercial use is prohibited.
If your contribution depends on third-party code or media, ensure that
its license is compatible (permissive: MIT/BSD/Apache-2.0; copyleft is
OK as long as you flag it).

## Building and testing

### TypeScript tree

```bash
bun install
bun run dev
bun run version
```

There is no automated test suite. Validate by booting the CLI and
exercising the path you changed.

### Go tree

```bash
cd go
go build -o bin/claudecode ./cmd/claudecode
go test ./...
go vet ./...
```

CI runs vet + build + test on every push (`.github/workflows/ci.yml`).

## Commit messages

Follow the existing style: short subject (`fix: …`, `feat: …`,
`refactor: …`, `docs: …`), wrap body at ~72 chars, explain *why*.

## What we will and will not accept

We will: bug fixes, new tools, new commands, tests, docs, performance
fixes, accessibility fixes, additional language support for the LSP
router.

We will not: anything that requires uploading to or scraping
proprietary services without authorization; PRs that introduce
opaque telemetry; PRs that add commercial features without a
licensing discussion first.

## Responsible disclosure

If you find a security issue, do not open a public issue. Email the
maintainer (see GitHub profile) or use a GitHub Security Advisory.

## Code style

- TypeScript: follow `tsconfig.json` + the existing patterns in `src/`. No semicolons, single quotes, camelCase.
- Go: gofmt-clean; minimal comments; stdlib-first; no third-party deps unless justified.

## Architectural decisions

For non-trivial work (new package, new public API, new external
dependency), open an issue first to discuss. Single-file fixes can
go straight to PR.
