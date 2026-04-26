package commands

import (
	"context"

	"claudecode/internal/core"
)

type jsonCmd struct{}

func NewJSON() core.Command { return &jsonCmd{} }

func (jsonCmd) Name() string     { return "json" }
func (jsonCmd) Synopsis() string { return "Toggle JSON output mode (notice only)" }

func (jsonCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo, "JSON output mode toggled (notice only; not yet wired to renderer).")
	return nil
}
