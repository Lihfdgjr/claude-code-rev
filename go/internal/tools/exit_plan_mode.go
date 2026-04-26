package tools

import (
	"context"
	"encoding/json"

	"claudecode/internal/chat"
	"claudecode/internal/core"
)

type exitPlanModeTool struct{}

func NewExitPlanMode() core.Tool { return &exitPlanModeTool{} }

func (t *exitPlanModeTool) Name() string { return "ExitPlanMode" }

func (t *exitPlanModeTool) Description() string {
	return "Exit plan mode. Mutating tools become callable again."
}

func (t *exitPlanModeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)
}

func (t *exitPlanModeTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	chat.PlanModeActive.Store(false)
	return "Plan mode disabled. Mutating tools are allowed again.", nil
}
