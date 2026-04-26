package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"claudecode/internal/core"
)

type todoItem struct {
	ID         string `json:"id,omitempty"`
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"activeForm,omitempty"`
}

var (
	todoMu   sync.Mutex
	todoList []todoItem
)

type todoWriteTool struct{}

func NewTodoWrite() core.Tool { return &todoWriteTool{} }

func (t *todoWriteTool) Name() string { return "TodoWrite" }

func (t *todoWriteTool) Description() string {
	return "Maintain a process-local todo list. Pass the full updated list each call. Status is one of pending, in_progress, completed."
}

func (t *todoWriteTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"todos": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"id": {"type": "string"},
						"content": {"type": "string"},
						"status": {"type": "string", "enum": ["pending", "in_progress", "completed"]},
						"activeForm": {"type": "string"}
					},
					"required": ["content", "status"],
					"additionalProperties": false
				}
			}
		},
		"required": ["todos"],
		"additionalProperties": false
	}`)
}

func (t *todoWriteTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Todos []todoItem `json:"todos"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	for i, td := range args.Todos {
		switch td.Status {
		case "pending", "in_progress", "completed":
		default:
			return "", fmt.Errorf("todo %d: invalid status %q", i, td.Status)
		}
		if strings.TrimSpace(td.Content) == "" {
			return "", fmt.Errorf("todo %d: empty content", i)
		}
	}

	todoMu.Lock()
	todoList = append(todoList[:0], args.Todos...)
	snapshot := append([]todoItem(nil), todoList...)
	todoMu.Unlock()

	var b strings.Builder
	for _, td := range snapshot {
		var box string
		switch td.Status {
		case "completed":
			box = "[x]"
		case "in_progress":
			box = "[~]"
		default:
			box = "[ ]"
		}
		text := td.Content
		if td.Status == "in_progress" && td.ActiveForm != "" {
			text = td.ActiveForm
		}
		fmt.Fprintf(&b, "%s %s\n", box, text)
	}
	if b.Len() == 0 {
		return "(no todos)", nil
	}
	return strings.TrimRight(b.String(), "\n"), nil
}
