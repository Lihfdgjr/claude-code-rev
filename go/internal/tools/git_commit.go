package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"claudecode/internal/core"
)

type gitCommitTool struct{}

type gitCommitInput struct {
	Message string `json:"message"`
	All     bool   `json:"all,omitempty"`
	Amend   bool   `json:"amend,omitempty"`
}

func NewGitCommit() core.Tool { return &gitCommitTool{} }

func (gitCommitTool) Name() string { return "GitCommit" }

func (gitCommitTool) Description() string {
	return "Create a git commit with the given message. Supports -a (stage tracked changes) and --amend."
}

func (gitCommitTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "message": {"type": "string"},
    "all": {"type": "boolean", "description": "Pass -a to stage all tracked modifications"},
    "amend": {"type": "boolean", "description": "Pass --amend to rewrite the previous commit"}
  },
  "required": ["message"],
  "additionalProperties": false
}`)
}

func (gitCommitTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in gitCommitInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if in.Message == "" {
		return "", fmt.Errorf("message is required")
	}

	args := []string{"commit", "-m", in.Message}
	if in.All {
		args = append(args, "-a")
	}
	if in.Amend {
		args = append(args, "--amend")
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return buf.String(), fmt.Errorf("git commit: %w", err)
	}
	return buf.String(), nil
}
