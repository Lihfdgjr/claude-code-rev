package commands

import (
	"context"

	"claudecode/internal/core"
)

type reviewCmd struct{}

func NewReview() core.Command { return &reviewCmd{} }

func (reviewCmd) Name() string     { return "review" }
func (reviewCmd) Synopsis() string { return "Review the current branch" }

func (reviewCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo,
		"Pending review of current branch. (Full implementation will run a series of checks.)")
	return nil
}
