package commands

import (
	"context"

	"claudecode/internal/core"
)

type syncCmd struct{}

func NewSync() core.Command { return &syncCmd{} }

func (syncCmd) Name() string     { return "sync" }
func (syncCmd) Synopsis() string { return "Sync settings (not configured)" }

func (syncCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo, "Settings sync requires a remote endpoint; not configured.")
	return nil
}
