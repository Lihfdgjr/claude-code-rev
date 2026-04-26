package commands

import (
	"context"

	"claudecode/internal/core"
)

type newCmd struct{}

func NewNew() core.Command { return &newCmd{} }

func (newCmd) Name() string     { return "new" }
func (newCmd) Synopsis() string { return "Start a fresh session (alias for /clear)" }

func (newCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.ResetHistory()
	sess.Notify(core.NotifyInfo, "Started a new session.")
	return nil
}
