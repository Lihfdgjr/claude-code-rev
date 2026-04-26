package commands

import (
	"context"

	"claudecode/internal/core"
)

type computerUseCmd struct{}

// NewComputerUseCmd returns the /computer-use slash command. The constructor
// is named to avoid clashing with tools.NewComputerUse.
func NewComputerUseCmd() core.Command { return &computerUseCmd{} }

func (computerUseCmd) Name() string     { return "computer-use" }
func (computerUseCmd) Synopsis() string { return "Show computer-use availability" }

func (computerUseCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo,
		"Computer use is not supported in this build. The ComputerUse tool surfaces the API but every action returns 'not supported'.")
	return nil
}
