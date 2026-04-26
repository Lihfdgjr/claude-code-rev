package commands

import (
	"context"

	"claudecode/internal/core"
)

type releaseNotesCmd struct{}

func NewReleaseNotes() core.Command { return &releaseNotesCmd{} }

func (releaseNotesCmd) Name() string     { return "release-notes" }
func (releaseNotesCmd) Synopsis() string { return "Show release notes" }

func (releaseNotesCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo,
		"claudecode-go 0.1.0 - initial Go implementation. See README for caveats.")
	return nil
}
