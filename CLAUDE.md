# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A restored Claude Code source tree reconstructed from source maps. It is **not** the original upstream repository. Some modules contain restoration-time fallbacks or shims where originals were unrecoverable. When making changes, prefer minimal auditable edits and document any workaround added because a module was restored with fallback or shim behavior.

## Development Commands

Requires **Bun 1.3.5+** and **Node.js 24+**.

```bash
bun install          # Install dependencies (includes local shim packages)
bun run dev          # Start the CLI via restored bootstrap entry
bun run start        # Alias for dev
bun run version      # Print CLI version (smoke test)
bun run dev:restore-check  # Run via the legacy dev-entry shim
```

There is no automated test suite or lint script. Validate changes by booting the CLI (`bun run dev`), running `bun run version`, and exercising the specific path you changed.

## Coding Style

- TypeScript with ESM imports, `react-jsx` (tsconfig target: ESNext, module: ESNext, strict: false)
- Omit semicolons, use single quotes, camelCase for variables/functions, PascalCase for React components and manager classes, kebab-case for command folder names (e.g. `src/commands/install-slack-app/`)
- Do not reorder imports when comments warn against it
- Path alias: `src/*` maps to `./src/*`

## Architecture

### Entry Point Flow

```
src/bootstrap-entry.ts          → checks special flags (--version, --daemon-worker, etc.)
  ↓
src/entrypoints/cli.tsx         → CLI-specific setup
  ↓
src/main.tsx                    → full initialization: configs, feature flags, OAuth,
                                  MCP servers, skills, model selection, React/Ink render loop
  ↓
src/commands.ts                 → command registry (110+ commands)
src/QueryEngine.ts              → core conversation loop (tool use, permissions, messages)
```

### Key Directories

| Directory | Purpose |
|-----------|---------|
| `src/tools/` | 51 tool implementations, each in its own subdirectory (BashTool, FileReadTool, AgentTool, etc.) |
| `src/commands/` | 110+ slash commands, each in its own subdirectory with descriptor objects |
| `src/services/` | Infrastructure: API client (`api/`), MCP protocol (`mcp/`), analytics (`analytics/`), session compaction (`compact/`), plugins (`plugins/`), LSP (`lsp/`), OAuth (`oauth/`), tool orchestration (`tools/`) |
| `src/components/` | 150+ React/Ink terminal UI components |
| `src/hooks/` | 80+ React hooks |
| `src/utils/` | 150+ utility modules |
| `src/state/` | Centralized AppState store |
| `src/context/` | React context providers |
| `src/ink/` | Custom enhanced Ink rendering pipeline (not standard ink) |
| `src/skills/` | Bundled skills (`claude-api`, `verify`) with reference docs |
| `src/entrypoints/` | Multiple entry points: CLI, MCP server, Agent SDK, sandbox |
| `vendor/` | Native dependency sources (audio-capture, image-processor, modifiers-napi, url-handler) |
| `shims/` | Compatibility shim packages for native/private modules that couldn't be restored from source maps |

### Tool System

Tools live in `src/tools/<ToolName>/` and define input/output schemas via Zod. They go through permission validation before execution. Tools can be built-in or provided by MCP servers. Tool orchestration and execution lives in `src/services/tools/`.

### UI Layer

The terminal UI uses a custom fork of Ink (React for terminals) in `src/ink/`. Key components:
- `VirtualMessageList` — virtual scrolling for large conversations
- `PromptInput/` — user input with typeahead (`useTypeahead.tsx`)
- `ScrollKeybindingHandler` — keyboard input routing
- `Message`/`Messages` — message rendering with diff, code highlighting
- `TrustDialog/` — permission prompts

### State & Sessions

- `src/state/AppState.tsx` — central state store
- Sessions persist to `~/.claude/` with transcript recording, recovery, and history search
- Memory system: CLAUDE.md parsing, short-term session memory, auto-summarization (`autoDream`)

### Shims & Restoration

The `shims/` directory contains local npm packages (referenced via `file:` in package.json) that provide fallback implementations for native bindings and private packages:
- `ant-claude-for-chrome-mcp` — Chrome MCP integration
- `ant-computer-use-*` — Computer use protocol
- `color-diff-napi`, `modifiers-napi`, `url-handler-napi` — native binding wrappers
