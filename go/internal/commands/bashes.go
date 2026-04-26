package commands

import (
	"context"

	"claudecode/internal/core"
)

type bashesCmd struct{}

func NewBashes() core.Command { return &bashesCmd{} }

func (bashesCmd) Name() string     { return "bashes" }
func (bashesCmd) Synopsis() string { return "Manage background bash jobs" }

func (bashesCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo, "Use the BashOutput and KillBash tools to manage background bash jobs.")
	return nil
}
