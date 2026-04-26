package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"claudecode/internal/core"
)

type bashOutputTool struct{}

func NewBashOutput() core.Tool { return &bashOutputTool{} }

func (t *bashOutputTool) Name() string { return "BashOutput" }

func (t *bashOutputTool) Description() string {
	return "Read incremental stdout/stderr from a background bash job started with run_in_background. Returns only new output since the previous call, plus current status and exit code if known."
}

func (t *bashOutputTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"bash_id": {"type": "string"},
			"filter": {"type": "string"}
		},
		"required": ["bash_id"],
		"additionalProperties": false
	}`)
}

func (t *bashOutputTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		BashID string `json:"bash_id"`
		Filter string `json:"filter"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if args.BashID == "" {
		return "", fmt.Errorf("bash_id required")
	}
	j, ok := lookupJob(args.BashID)
	if !ok {
		return "", fmt.Errorf("no background job with id %q", args.BashID)
	}

	status := "running"
	exit := -1
	if j.Done {
		if j.Err != nil {
			status = "failed"
		} else {
			status = "exited"
		}
		exit = j.ExitCode
	}

	chunk := string(drainNew(j))
	if args.Filter != "" {
		re, err := regexp.Compile(args.Filter)
		if err != nil {
			return "", fmt.Errorf("filter regex: %w", err)
		}
		var kept []string
		for _, line := range strings.Split(chunk, "\n") {
			if re.MatchString(line) {
				kept = append(kept, line)
			}
		}
		chunk = strings.Join(kept, "\n")
	}

	header := fmt.Sprintf("status=%s exit=%d", status, exit)
	if chunk == "" {
		return header, nil
	}
	return header + "\n" + chunk, nil
}
