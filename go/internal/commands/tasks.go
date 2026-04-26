package commands

import (
	"context"

	"claudecode/internal/core"
)

type tasksCmd struct{}

func NewTasks() core.Command { return &tasksCmd{} }

func (tasksCmd) Name() string     { return "tasks" }
func (tasksCmd) Synopsis() string { return "Show task management hint" }

func (tasksCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo, "Task management is exposed via TodoWrite tool and TaskCreate (when wired).")
	return nil
}
