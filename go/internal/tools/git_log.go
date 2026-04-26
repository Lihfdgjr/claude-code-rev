package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"

	"claudecode/internal/core"
)

type gitLogTool struct{}

type gitLogInput struct {
	Path    string `json:"path,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Oneline *bool  `json:"oneline,omitempty"`
}

func NewGitLog() core.Tool { return &gitLogTool{} }

func (gitLogTool) Name() string { return "GitLog" }

func (gitLogTool) Description() string {
	return "Run 'git log' with optional path filter and limit. Defaults to --oneline -n 10."
}

func (gitLogTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "path": {"type": "string"},
    "limit": {"type": "integer", "minimum": 1, "description": "Max commits (default 10)"},
    "oneline": {"type": "boolean", "description": "If true (default), use --oneline"}
  },
  "additionalProperties": false
}`)
}

func (gitLogTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in gitLogInput
	if len(input) > 0 {
		if err := json.Unmarshal(input, &in); err != nil {
			return "", fmt.Errorf("invalid input: %w", err)
		}
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}
	oneline := true
	if in.Oneline != nil {
		oneline = *in.Oneline
	}

	args := []string{"log"}
	if oneline {
		args = append(args, "--oneline")
	}
	args = append(args, "-n", strconv.Itoa(limit))
	if in.Path != "" {
		args = append(args, "--", in.Path)
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return buf.String(), fmt.Errorf("git log: %w", err)
	}
	return buf.String(), nil
}
