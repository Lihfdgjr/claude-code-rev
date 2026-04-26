package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type taskListTool struct{}

func NewTaskList() core.Tool { return &taskListTool{} }

func (t *taskListTool) Name() string { return "TaskList" }

func (t *taskListTool) Description() string {
	return "List all tasks in the in-process registry."
}

func (t *taskListTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)
}

func (t *taskListTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	tasks := listTasks()
	if len(tasks) == 0 {
		return "(no tasks)", nil
	}

	idW, statusW := len("id"), len("status")
	for _, t := range tasks {
		if w := len(fmt.Sprintf("%d", t.ID)); w > idW {
			idW = w
		}
		if w := len(t.Status); w > statusW {
			statusW = w
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%-*s | %-*s | %s\n", idW, "id", statusW, "status", "subject")
	fmt.Fprintf(&b, "%s-+-%s-+-%s\n", strings.Repeat("-", idW), strings.Repeat("-", statusW), strings.Repeat("-", len("subject")))
	for _, t := range tasks {
		fmt.Fprintf(&b, "%-*d | %-*s | %s\n", idW, t.ID, statusW, t.Status, t.Subject)
	}
	return strings.TrimRight(b.String(), "\n"), nil
}
