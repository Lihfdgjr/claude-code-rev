package commands

import (
	"context"
	"fmt"

	"claudecode/internal/core"
)

type redoCmd struct{}

func NewRedo() core.Command { return &redoCmd{} }

func (redoCmd) Name() string     { return "redo" }
func (redoCmd) Synopsis() string { return "Redo the last undone action" }

func (redoCmd) Run(ctx context.Context, args string, sess core.Session) error {
	label, ok := sess.Redo()
	if !ok {
		sess.Notify(core.NotifyInfo, "Nothing to redo.")
		return nil
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Redid: %s.", label))
	return nil
}
