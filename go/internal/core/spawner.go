package core

import "context"

type SpawnOptions struct {
	Description  string
	Prompt       string
	SystemPrompt string
	Model        string
	MaxTurns     int
	Tools        []Tool
	// AllowedTools, when non-empty, restricts the spawner to only the
	// named tools out of its own registry. Used when the caller does not
	// have access to the tool objects directly (e.g. agent.go).
	AllowedTools []string
}

type Spawner interface {
	Spawn(ctx context.Context, opts SpawnOptions) (string, error)
}
