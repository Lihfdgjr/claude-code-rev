package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"claudecode/internal/core"
)

type taskUpdateTool struct{}

func NewTaskUpdate() core.Tool { return &taskUpdateTool{} }

func (t *taskUpdateTool) Name() string { return "TaskUpdate" }

func (t *taskUpdateTool) Description() string {
	return "Update an existing task by id. Status may be pending, in_progress, completed, or deleted (removes the task)."
}

func (t *taskUpdateTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"taskId": {"type": ["integer", "string"]},
			"status": {"type": "string", "enum": ["pending", "in_progress", "completed", "deleted"]},
			"subject": {"type": "string"},
			"description": {"type": "string"}
		},
		"required": ["taskId"],
		"additionalProperties": false
	}`)
}

func (t *taskUpdateTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var raw struct {
		TaskID      json.RawMessage `json:"taskId"`
		Status      *string         `json:"status"`
		Subject     *string         `json:"subject"`
		Description *string         `json:"description"`
	}
	if err := json.Unmarshal(input, &raw); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	id, err := parseTaskID(raw.TaskID)
	if err != nil {
		return "", err
	}

	if raw.Status != nil {
		switch *raw.Status {
		case "pending", "in_progress", "completed":
		case "deleted":
			if !removeTask(id) {
				return "", fmt.Errorf("task #%d not found", id)
			}
			return fmt.Sprintf("deleted task #%d", id), nil
		default:
			return "", fmt.Errorf("invalid status %q", *raw.Status)
		}
	}

	updated := updateTask(id, func(e *taskEntry) {
		if raw.Status != nil {
			e.Status = *raw.Status
		}
		if raw.Subject != nil {
			e.Subject = *raw.Subject
		}
		if raw.Description != nil {
			e.Description = *raw.Description
		}
	})
	if updated == nil {
		return "", fmt.Errorf("task #%d not found", id)
	}
	return fmt.Sprintf("task #%d: %s [status=%s]", updated.ID, updated.Subject, updated.Status), nil
}

func parseTaskID(raw json.RawMessage) (int, error) {
	if len(raw) == 0 {
		return 0, fmt.Errorf("taskId is required")
	}
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		v, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("invalid taskId %q", s)
		}
		return v, nil
	}
	return 0, fmt.Errorf("taskId must be int or string")
}
