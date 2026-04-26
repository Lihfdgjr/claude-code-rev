package tools

import (
	"context"
	"encoding/json"

	"claudecode/internal/chat"
	"claudecode/internal/core"
)

type enterPlanModeTool struct{}

func NewEnterPlanMode() core.Tool { return &enterPlanModeTool{} }

func (t *enterPlanModeTool) Name() string { return "EnterPlanMode" }

func (t *enterPlanModeTool) Description() string {
	return "Enter plan mode. While active, mutating tools (Write/Edit/Bash/...) are blocked."
}

func (t *enterPlanModeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)
}

func (t *enterPlanModeTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	chat.PlanModeActive.Store(true)
	return "Plan mode is now active. Mutating tools will be blocked until ExitPlanMode is invoked.", nil
}
