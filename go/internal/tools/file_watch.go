package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"claudecode/internal/core"
)

type fileWatchTool struct{}

type fileWatchInput struct {
	Path    string `json:"path"`
	Seconds int    `json:"seconds,omitempty"`
}

func NewFileWatch() core.Tool { return &fileWatchTool{} }

func (fileWatchTool) Name() string { return "FileWatch" }

func (fileWatchTool) Description() string {
	return "Poll a file's mtime and size every second for up to N seconds (default 30, max 300). Returns 'unchanged' on timeout or 'changed at <RFC3339>: <new size>' on first change."
}

func (fileWatchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "path": {"type": "string"},
    "seconds": {"type": "integer", "minimum": 1, "maximum": 300, "description": "Watch duration (default 30, max 300)"}
  },
  "required": ["path"],
  "additionalProperties": false
}`)
}

func (fileWatchTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in fileWatchInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if in.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	secs := in.Seconds
	if secs <= 0 {
		secs = 30
	}
	if secs > 300 {
		secs = 300
	}

	fi, err := os.Stat(in.Path)
	if err != nil {
		return "", fmt.Errorf("stat: %w", err)
	}
	startMtime := fi.ModTime()
	startSize := fi.Size()

	deadline := time.Now().Add(time.Duration(secs) * time.Second)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			cur, err := os.Stat(in.Path)
			if err != nil {
				return fmt.Sprintf("changed at %s: stat error %v", time.Now().UTC().Format(time.RFC3339), err), nil
			}
			if !cur.ModTime().Equal(startMtime) || cur.Size() != startSize {
				return fmt.Sprintf("changed at %s: %d", time.Now().UTC().Format(time.RFC3339), cur.Size()), nil
			}
			if time.Now().After(deadline) {
				return "unchanged", nil
			}
		}
	}
}
