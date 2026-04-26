# claudecode-go

Independent Go implementation of a Claude Code style CLI. Written from scratch — shares no code with the TypeScript tree above.

## Build

```
cd go
go build -o bin/claudecode ./cmd/claudecode
./bin/claudecode version
```

## Run

```
export ANTHROPIC_API_KEY=sk-...
./bin/claudecode
```

## Layout

| Path | Purpose |
|------|---------|
| `cmd/claudecode/` | Entry point |
| `internal/core/` | Shared types and interfaces (Tool, Command, Session, Transport) |
| `internal/api/` | Anthropic Messages API client + SSE stream |
| `internal/chat/` | Conversation session and agentic loop |
| `internal/tools/` | Built-in tools: Read, Write, Edit, Bash, Grep, Glob, LS |
| `internal/commands/` | Slash commands: /help /clear /model /compact /init /memory |
| `internal/ui/` | Bubble Tea terminal UI |
| `internal/config/` | Config loader (`~/.claude` + env) |
| `internal/memory/` | CLAUDE.md discovery and parsing |
| `internal/permissions/` | Tool permission gate |
| `internal/util/` | Small shared helpers |
