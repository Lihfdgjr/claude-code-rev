package commands

import (
	"context"

	"claudecode/internal/core"
)

type cancelCmd struct{}

// NewCancel returns a /cancel command that aborts the in-flight turn
// by invoking Session.Cancel (wired to Driver.Cancel by NewDriver).
func NewCancel() core.Command { return &cancelCmd{} }

func (cancelCmd) Name() string     { return "cancel" }
func (cancelCmd) Synopsis() string { return "Cancel the in-flight turn" }

func (cancelCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Cancel()
	sess.Notify(core.NotifyInfo, "cancelled.")
	return nil
}
