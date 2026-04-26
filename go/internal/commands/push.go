package commands

import (
	"context"

	"claudecode/internal/core"
)

type pushCmd struct{}

func NewPush() core.Command { return &pushCmd{} }

func (pushCmd) Name() string     { return "push" }
func (pushCmd) Synopsis() string { return "Suggest a git push command" }

func (pushCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo, "Use the Bash tool to run: git push")
	return nil
}
