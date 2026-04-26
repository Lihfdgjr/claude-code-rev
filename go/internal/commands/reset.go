package commands

import (
	"context"

	"claudecode/internal/core"
)

type resetCmd struct{}

func NewReset() core.Command { return &resetCmd{} }

func (resetCmd) Name() string     { return "reset" }
func (resetCmd) Synopsis() string { return "Alias for /clear" }

func (resetCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Checkpoint("before /reset")
	sess.ResetHistory()
	sess.Notify(core.NotifyInfo, "Conversation history cleared.")
	return nil
}
