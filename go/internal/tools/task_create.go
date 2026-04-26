package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type taskCreateTool struct{}

func NewTaskCreate() core.Tool { return &taskCreateTool{} }

func (t *taskCreateTool) Name() string { return "TaskCreate" }

func (t *taskCreateTool) Description() string {
	return "Create a new task in the in-process task registry. Returns the assigned task id."
}

func (t *taskCreateTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"subject": {"type": "string"},
			"description": {"type": "string"},
			"activeForm": {"type": "string"}
		},
		"required": ["subject"],
		"additionalProperties": false
	}`)
}

func (t *taskCreateTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Subject     string `json:"subject"`
		Description string `json:"description"`
		ActiveForm  string `json:"activeForm"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if strings.TrimSpace(args.Subject) == "" {
		return "", errors.New("subject is required")
	}
	task := addTask(args.Subject, args.Description, args.ActiveForm)
	return fmt.Sprintf("created task #%d: %s [status=%s]", task.ID, task.Subject, task.Status), nil
}
