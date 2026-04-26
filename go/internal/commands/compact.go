package commands

import (
	"context"
	"fmt"

	"claudecode/internal/core"
)

type compactCmd struct{}

func NewCompact() core.Command { return &compactCmd{} }

func (compactCmd) Name() string     { return "compact" }
func (compactCmd) Synopsis() string { return "Summarize and compact the conversation" }

func (compactCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Checkpoint("before /compact")
	if err := sess.Compact(ctx); err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("Compact failed: %v", err))
		return err
	}
	sess.Notify(core.NotifyInfo, "Conversation compacted.")
	return nil
}
