package commands

import (
	"context"
	"fmt"

	"claudecode/internal/core"
)

type summaryCmd struct{}

func NewSummary() core.Command { return &summaryCmd{} }

func (summaryCmd) Name() string     { return "summary" }
func (summaryCmd) Synopsis() string { return "Summarize the conversation (alias for /compact)" }

func (summaryCmd) Run(ctx context.Context, args string, sess core.Session) error {
	if err := sess.Compact(ctx); err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("Summary failed: %v", err))
		return err
	}
	sess.Notify(core.NotifyInfo, "Conversation summarized.")
	return nil
}
