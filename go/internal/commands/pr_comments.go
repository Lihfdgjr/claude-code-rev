package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type prCommentsCmd struct{}

func NewPRComments() core.Command { return &prCommentsCmd{} }

func (prCommentsCmd) Name() string     { return "pr_comments" }
func (prCommentsCmd) Synopsis() string { return "Show how to fetch PR review comments" }

func (prCommentsCmd) Run(ctx context.Context, args string, sess core.Session) error {
	pr := strings.TrimSpace(args)
	if pr == "" {
		pr = "<pr>"
	}
	sess.Notify(core.NotifyInfo,
		fmt.Sprintf("Use the Bash tool to run `gh api repos/OWNER/REPO/pulls/%s/comments`.", pr))
	return nil
}
