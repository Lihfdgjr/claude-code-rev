package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"claudecode/internal/agents"
	"claudecode/internal/core"
)

type agentTool struct {
	spawner core.Spawner
	defs    *agents.Registry
}

type agentInput struct {
	Description  string `json:"description"`
	Prompt       string `json:"prompt"`
	SubagentType string `json:"subagent_type,omitempty"`
}

func NewAgent(spawner core.Spawner, defs *agents.Registry) core.Tool {
	return &agentTool{spawner: spawner, defs: defs}
}

func (a *agentTool) Name() string { return "Agent" }

func (a *agentTool) Description() string {
	return "Spawn a sub-agent with a focused prompt to research, analyze, or perform a self-contained task. Returns the sub-agent's final response."
}

func (a *agentTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "description": {"type": "string", "description": "3-5 word task description"},
    "prompt": {"type": "string", "description": "Full briefing for the sub-agent"},
    "subagent_type": {"type": "string", "description": "Sub-agent flavor", "default": "general-purpose"}
  },
  "required": ["description", "prompt"],
  "additionalProperties": false
}`)
}

const generalPurposeSystemPrompt = "You are a general-purpose sub-agent. You have been spawned to investigate or carry out a focused task on behalf of the parent agent. Use the tools available to you to gather information, perform the work, and verify your results. Be thorough but stay on task. When you finish, return a concise final response that the parent agent can act on directly; do not include filler, plans, or status updates beyond what the parent needs."

func (a *agentTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	if a.spawner == nil {
		return "", fmt.Errorf("subagent runner not configured")
	}

	var in agentInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if strings.TrimSpace(in.Description) == "" {
		return "", fmt.Errorf("description is required")
	}
	if strings.TrimSpace(in.Prompt) == "" {
		return "", fmt.Errorf("prompt is required")
	}
	subtype := in.SubagentType
	if subtype == "" {
		subtype = "general-purpose"
	}

	var def *agents.Definition
	if a.defs != nil {
		if d, ok := a.defs.Get(subtype); ok {
			def = d
		} else if d, ok := a.defs.Get("general-purpose"); ok {
			def = d
		}
	}

	systemPrompt := generalPurposeSystemPrompt
	var model string
	var maxTurns int
	var allowedTools []string
	if def != nil {
		if strings.TrimSpace(def.SystemPrompt) != "" {
			systemPrompt = def.SystemPrompt
		}
		model = def.Model
		maxTurns = def.MaxTurns
		if len(def.AllowedTools) > 0 {
			allowedTools = append(allowedTools, def.AllowedTools...)
		}
	}

	opts := core.SpawnOptions{
		Description:  in.Description,
		Prompt:       in.Prompt,
		SystemPrompt: systemPrompt,
		Model:        model,
		MaxTurns:     maxTurns,
		Tools:        nil,
		AllowedTools: allowedTools,
	}

	out, err := a.spawner.Spawn(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("subagent failed: %w", err)
	}
	return out, nil
}
