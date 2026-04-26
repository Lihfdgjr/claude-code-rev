package commands

import (
	"context"

	"claudecode/internal/core"
)

type bugCmd struct{}

func NewBug() core.Command { return &bugCmd{} }

func (bugCmd) Name() string     { return "bug" }
func (bugCmd) Synopsis() string { return "Show how to file a bug report" }

func (bugCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo,
		"Report bugs at https://github.com/anthropics/claude-code/issues - include claudecode-go and your OS.")
	return nil
}
