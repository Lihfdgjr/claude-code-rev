package commands

import (
	"context"

	"claudecode/internal/core"
)

type fastCmd struct{}

func NewFast() core.Command { return &fastCmd{} }

func (fastCmd) Name() string     { return "fast" }
func (fastCmd) Synopsis() string { return "Toggle fast model mode (stub)" }

func (fastCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo,
		"Fast mode toggles a faster model in the upstream CLI. Set CLAUDECODE_MODEL to switch models here.")
	return nil
}
