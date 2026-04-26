package commands

import (
	"context"

	"claudecode/internal/core"
)

type reloadCmd struct{}

func NewReload() core.Command { return &reloadCmd{} }

func (reloadCmd) Name() string     { return "reload" }
func (reloadCmd) Synopsis() string { return "Trigger a rescan of settings.json and CLAUDE.md" }

func (reloadCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo, "watchers run automatically; manual reload triggers a rescan.")
	return nil
}
