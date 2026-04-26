package commands

import (
	"context"

	"claudecode/internal/core"
)

type permissionsCmd struct{}

func NewPermissions() core.Command { return &permissionsCmd{} }

func (permissionsCmd) Name() string     { return "permissions" }
func (permissionsCmd) Synopsis() string { return "Show how to configure tool permissions" }

func (permissionsCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo,
		"Edit `permissions.mode` in ~/.claude/settings.json (allow|deny|ask). Add tool names to permissions.allow / permissions.deny arrays.")
	return nil
}
