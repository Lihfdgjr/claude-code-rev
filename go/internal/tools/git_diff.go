package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"claudecode/internal/core"
)

type gitDiffTool struct{}

type gitDiffInput struct {
	Path   string `json:"path,omitempty"`
	Staged bool   `json:"staged,omitempty"`
}

func NewGitDiff() core.Tool { return &gitDiffTool{} }

func (gitDiffTool) Name() string { return "GitDiff" }

func (gitDiffTool) Description() string {
	return "Run 'git diff' optionally with --cached and a path. Returns the combined output."
}

func (gitDiffTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "path": {"type": "string", "description": "Optional path filter"},
    "staged": {"type": "boolean", "description": "If true, diff staged changes (--cached)"}
  },
  "additionalProperties": false
}`)
}

func (gitDiffTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in gitDiffInput
	if len(input) > 0 {
		if err := json.Unmarshal(input, &in); err != nil {
			return "", fmt.Errorf("invalid input: %w", err)
		}
	}
	args := []string{"diff"}
	if in.Staged {
		args = append(args, "--cached")
	}
	if in.Path != "" {
		args = append(args, "--", in.Path)
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return buf.String(), fmt.Errorf("git diff: %w", err)
	}
	return buf.String(), nil
}
