package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"claudecode/internal/core"
)

type killBashTool struct{}

func NewKillBash() core.Tool { return &killBashTool{} }

func (t *killBashTool) Name() string { return "KillBash" }

func (t *killBashTool) Description() string {
	return "Terminate a background bash job by id. Sends SIGTERM (or kill on Windows), waits up to 2s, then force kills."
}

func (t *killBashTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"bash_id": {"type": "string"}
		},
		"required": ["bash_id"],
		"additionalProperties": false
	}`)
}

func (t *killBashTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		BashID string `json:"bash_id"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if args.BashID == "" {
		return "", fmt.Errorf("bash_id required")
	}
	j, ok := lookupJob(args.BashID)
	if !ok {
		return "not found", nil
	}

	if j.Done {
		removeJob(args.BashID)
		return fmt.Sprintf("killed %s", args.BashID), nil
	}

	_ = killJob(j)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if j.Done {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !j.Done && j.Cmd != nil && j.Cmd.Process != nil {
		_ = j.Cmd.Process.Kill()
	}

	removeJob(args.BashID)
	return fmt.Sprintf("killed %s", args.BashID), nil
}
