package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type attachmentsCmd struct{}

func NewAttachments() core.Command { return &attachmentsCmd{} }

func (attachmentsCmd) Name() string     { return "attachments" }
func (attachmentsCmd) Synopsis() string { return "List pending attachments for the next message" }

func (attachmentsCmd) Run(ctx context.Context, args string, sess core.Session) error {
	all := sess.DrainAttachments()
	for _, b := range all {
		sess.Attach(b)
	}
	if len(all) == 0 {
		sess.Notify(core.NotifyInfo, "no attachments pending")
		return nil
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("%d attachment(s) pending: %s", len(all), describeAttachments(all)))
	return nil
}

func describeAttachments(blocks []core.Block) string {
	counts := map[core.BlockKind]int{}
	for _, b := range blocks {
		counts[b.Kind()]++
	}
	parts := make([]string, 0, len(counts))
	for k, n := range counts {
		parts = append(parts, fmt.Sprintf("%s x%d", k, n))
	}
	return strings.Join(parts, ", ")
}
