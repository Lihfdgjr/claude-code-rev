package commands

import (
	"context"

	"claudecode/internal/chat"
	"claudecode/internal/core"
)

type thinkingCmd struct{}

func NewThinking() core.Command { return &thinkingCmd{} }

func (thinkingCmd) Name() string     { return "thinking" }
func (thinkingCmd) Synopsis() string { return "toggle extended thinking mode" }

func (thinkingCmd) Run(ctx context.Context, args string, sess core.Session) error {
	cur := chat.ThinkingEnabled.Load()
	chat.ThinkingEnabled.Store(!cur)
	if !cur {
		sess.Notify(core.NotifyInfo, "Extended thinking: on")
	} else {
		sess.Notify(core.NotifyInfo, "Extended thinking: off")
	}
	return nil
}
