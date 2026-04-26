package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"claudecode/internal/core"
)

type worktreeCreateTool struct{}

func NewWorktreeCreate() core.Tool { return &worktreeCreateTool{} }

func (t *worktreeCreateTool) Name() string { return "WorktreeCreate" }

func (t *worktreeCreateTool) Description() string {
	return "Run `git worktree add <path>`. If 'branch' is provided, creates a new branch with `-b`."
}

func (t *worktreeCreateTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string"},
			"branch": {"type": "string"}
		},
		"required": ["path"],
		"additionalProperties": false
	}`)
}

func (t *worktreeCreateTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Path   string `json:"path"`
		Branch string `json:"branch"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if strings.TrimSpace(args.Path) == "" {
		return "", fmt.Errorf("path required")
	}

	cmdArgs := []string{"worktree", "add"}
	if args.Branch != "" {
		cmdArgs = append(cmdArgs, "-b", args.Branch)
	}
	cmdArgs = append(cmdArgs, args.Path)

	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		return "", fmt.Errorf("git worktree add failed: %w\n%s", err, output)
	}
	if output == "" {
		output = "(no output)"
	}
	return fmt.Sprintf("created worktree at %s\n%s", args.Path, output), nil
}
