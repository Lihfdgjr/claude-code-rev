package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"claudecode/internal/core"
)

type worktreeRemoveTool struct{}

func NewWorktreeRemove() core.Tool { return &worktreeRemoveTool{} }

func (t *worktreeRemoveTool) Name() string { return "WorktreeRemove" }

func (t *worktreeRemoveTool) Description() string {
	return "Run `git worktree remove <path>`. Adds `--force` when force=true."
}

func (t *worktreeRemoveTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string"},
			"force": {"type": "boolean"}
		},
		"required": ["path"],
		"additionalProperties": false
	}`)
}

func (t *worktreeRemoveTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Path  string `json:"path"`
		Force bool   `json:"force"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if strings.TrimSpace(args.Path) == "" {
		return "", fmt.Errorf("path required")
	}

	cmdArgs := []string{"worktree", "remove"}
	if args.Force {
		cmdArgs = append(cmdArgs, "--force")
	}
	cmdArgs = append(cmdArgs, args.Path)

	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		return "", fmt.Errorf("git worktree remove failed: %w\n%s", err, output)
	}
	if output == "" {
		output = "(removed)"
	}
	return fmt.Sprintf("removed worktree at %s\n%s", args.Path, output), nil
}
