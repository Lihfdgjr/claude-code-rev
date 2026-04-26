package commands

import (
	"context"

	"claudecode/internal/core"
)

type diffCmd struct{}

func NewDiff() core.Command { return &diffCmd{} }

func (diffCmd) Name() string     { return "diff" }
func (diffCmd) Synopsis() string { return "Suggest a git diff command" }

func (diffCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo, "Use the Bash tool to run: git diff (or git diff --cached)")
	return nil
}
