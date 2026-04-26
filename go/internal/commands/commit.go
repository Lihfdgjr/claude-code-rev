package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type commitCmd struct{}

func NewCommit() core.Command { return &commitCmd{} }

func (commitCmd) Name() string     { return "commit" }
func (commitCmd) Synopsis() string { return "Suggest a git commit command" }

func (commitCmd) Run(ctx context.Context, args string, sess core.Session) error {
	msg := strings.TrimSpace(args)
	if msg == "" {
		msg = "<your message>"
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Use the Bash tool to run: git commit -m '%s'", msg))
	return nil
}
