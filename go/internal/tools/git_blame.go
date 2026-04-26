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

type gitBlameTool struct{}

type gitBlameInput struct {
	File      string `json:"file"`
	LineStart int    `json:"line_start,omitempty"`
	LineEnd   int    `json:"line_end,omitempty"`
}

func NewGitBlame() core.Tool { return &gitBlameTool{} }

func (gitBlameTool) Name() string { return "GitBlame" }

func (gitBlameTool) Description() string {
	return "Run 'git blame' on a file, optionally limited to a line range."
}

func (gitBlameTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "file": {"type": "string"},
    "line_start": {"type": "integer", "minimum": 1},
    "line_end": {"type": "integer", "minimum": 1}
  },
  "required": ["file"],
  "additionalProperties": false
}`)
}

func (gitBlameTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in gitBlameInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if in.File == "" {
		return "", fmt.Errorf("file is required")
	}

	args := []string{"blame"}
	if in.LineStart > 0 && in.LineEnd > 0 {
		args = append(args, "-L", strconv.Itoa(in.LineStart)+","+strconv.Itoa(in.LineEnd))
	}
	args = append(args, in.File)

	cmd := exec.CommandContext(ctx, "git", args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return buf.String(), fmt.Errorf("git blame: %w", err)
	}
	return buf.String(), nil
}
