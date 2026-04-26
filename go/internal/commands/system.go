package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type systemCmd struct{}

func NewSystem() core.Command { return &systemCmd{} }

func (systemCmd) Name() string     { return "system" }
func (systemCmd) Synopsis() string { return "Show or set the system prompt" }

func (systemCmd) Run(ctx context.Context, args string, sess core.Session) error {
	text := strings.TrimSpace(args)
	if text != "" {
		sess.SetSystemPrompt(text)
		sess.Notify(core.NotifyInfo, "System prompt updated.")
		return nil
	}
	cur := sess.SystemPrompt()
	if cur == "" {
		sess.Notify(core.NotifyInfo, "(no system prompt set)")
		return nil
	}
	const limit = 2000
	if len(cur) > limit {
		cur = cur[:limit] + "\n... (truncated)"
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("System prompt:\n%s", cur))
	return nil
}
