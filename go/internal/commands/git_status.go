package commands

import (
	"context"

	"claudecode/internal/core"
)

type gitStatusCmd struct{}

func NewGitStatus() core.Command { return &gitStatusCmd{} }

func (gitStatusCmd) Name() string     { return "git-status" }
func (gitStatusCmd) Synopsis() string { return "Suggest a git status command" }

func (gitStatusCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo, "Use the Bash tool to run: git status")
	return nil
}
