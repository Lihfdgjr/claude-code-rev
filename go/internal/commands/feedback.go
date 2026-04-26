package commands

import (
	"context"

	"claudecode/internal/core"
)

type feedbackCmd struct{}

func NewFeedback() core.Command { return &feedbackCmd{} }

func (feedbackCmd) Name() string     { return "feedback" }
func (feedbackCmd) Synopsis() string { return "Alias for /bug" }

func (feedbackCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo,
		"Report bugs at https://github.com/anthropics/claude-code/issues - include claudecode-go and your OS.")
	return nil
}
