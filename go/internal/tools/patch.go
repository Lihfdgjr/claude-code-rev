package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"claudecode/internal/core"
)

type patchTool struct{}

type patchInput struct {
	Patch     string `json:"patch"`
	TargetDir string `json:"target_dir,omitempty"`
}

func NewPatch() core.Tool { return &patchTool{} }

func (patchTool) Name() string { return "Patch" }

func (patchTool) Description() string {
	return "Apply a unified diff. Tries 'git apply' first; on POSIX systems, falls back to 'patch -p1' if git apply fails."
}

func (patchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "patch": {"type": "string", "description": "Unified diff text"},
    "target_dir": {"type": "string", "description": "Directory to apply in (default cwd)"}
  },
  "required": ["patch"],
  "additionalProperties": false
}`)
}

func (patchTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in patchInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if in.Patch == "" {
		return "", fmt.Errorf("patch is required")
	}

	dir := in.TargetDir
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = cwd
	}

	tmp, err := os.CreateTemp("", "patch-*.diff")
	if err != nil {
		return "", fmt.Errorf("temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.WriteString(in.Patch); err != nil {
		tmp.Close()
		return "", fmt.Errorf("write patch: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close patch: %w", err)
	}

	gitCmd := exec.CommandContext(ctx, "git", "apply", tmpPath)
	gitCmd.Dir = dir
	var gitBuf bytes.Buffer
	gitCmd.Stdout = &gitBuf
	gitCmd.Stderr = &gitBuf
	if err := gitCmd.Run(); err == nil {
		return "git apply succeeded\n" + gitBuf.String(), nil
	}
	gitOut := gitBuf.String()

	patchCmd := exec.CommandContext(ctx, "patch", "-p1", "-i", tmpPath)
	patchCmd.Dir = dir
	var patchBuf bytes.Buffer
	patchCmd.Stdout = &patchBuf
	patchCmd.Stderr = &patchBuf
	if err := patchCmd.Run(); err != nil {
		return "git apply output:\n" + gitOut + "\npatch output:\n" + patchBuf.String(), fmt.Errorf("patch apply failed: %w", err)
	}
	return "patch -p1 succeeded\n" + patchBuf.String(), nil
}
