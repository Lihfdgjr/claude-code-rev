package core

import (
	"context"
	"encoding/json"
)

type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Run(ctx context.Context, input json.RawMessage) (string, error)
}

type ToolRegistry interface {
	Get(name string) (Tool, bool)
	All() []Tool
}
