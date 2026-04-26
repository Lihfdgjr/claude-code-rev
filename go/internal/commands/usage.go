package commands

import (
	"context"

	"claudecode/internal/core"
)

type usageCmd struct{}

func NewUsage() core.Command { return &usageCmd{} }

func (usageCmd) Name() string     { return "usage" }
func (usageCmd) Synopsis() string { return "Show usage hint (alias for /cost)" }

func (usageCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo, "See /cost for token usage and estimated session cost.")
	return nil
}
