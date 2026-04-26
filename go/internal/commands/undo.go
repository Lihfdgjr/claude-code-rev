package commands

import (
	"context"
	"fmt"

	"claudecode/internal/core"
)

type undoCmd struct{}

func NewUndo() core.Command { return &undoCmd{} }

func (undoCmd) Name() string     { return "undo" }
func (undoCmd) Synopsis() string { return "Undo the last history-changing action" }

func (undoCmd) Run(ctx context.Context, args string, sess core.Session) error {
	label, ok := sess.Undo()
	if !ok {
		sess.Notify(core.NotifyInfo, "Nothing to undo.")
		return nil
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Undid: %s.", label))
	return nil
}
