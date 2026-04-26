package commands

import (
	"context"
	"fmt"

	"claudecode/internal/core"
)

type clearAttachmentsCmd struct{}

func NewClearAttachments() core.Command { return &clearAttachmentsCmd{} }

func (clearAttachmentsCmd) Name() string     { return "clear-attachments" }
func (clearAttachmentsCmd) Synopsis() string { return "Discard all pending attachments" }

func (clearAttachmentsCmd) Run(ctx context.Context, args string, sess core.Session) error {
	all := sess.DrainAttachments()
	sess.Notify(core.NotifyInfo, fmt.Sprintf("cleared %d attachment(s)", len(all)))
	return nil
}
