package commands

import (
	"context"

	"claudecode/internal/core"
)

type clearCmd struct{}

func NewClear() core.Command { return &clearCmd{} }

func (clearCmd) Name() string     { return "clear" }
func (clearCmd) Synopsis() string { return "Clear conversation history" }

func (clearCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Checkpoint("before /clear")
	sess.ResetHistory()
	sess.Notify(core.NotifyInfo, "Conversation history cleared.")
	return nil
}
