package commands

import (
	"context"

	"claudecode/internal/core"
)

type timestampCmd struct{}

func NewTimestamp() core.Command { return &timestampCmd{} }

func (timestampCmd) Name() string     { return "timestamp" }
func (timestampCmd) Synopsis() string { return "Toggle timestamps in history (notice only)" }

func (timestampCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo, "Timestamps in history toggled (notice only; not yet wired to renderer).")
	return nil
}
