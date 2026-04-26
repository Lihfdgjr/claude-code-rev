package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type messagesCmd struct{}

func NewMessages() core.Command { return &messagesCmd{} }

func (messagesCmd) Name() string     { return "messages" }
func (messagesCmd) Synopsis() string { return "Compact one-line-per-message history view" }

func (messagesCmd) Run(ctx context.Context, args string, sess core.Session) error {
	hist := sess.History()
	if len(hist) == 0 {
		sess.Notify(core.NotifyInfo, "(no messages)")
		return nil
	}
	var b strings.Builder
	for i, m := range hist {
		var label string
		switch m.Role {
		case core.RoleUser:
			label = "USER"
		case core.RoleAssistant:
			label = "ASSIST"
		case core.RoleSystem:
			label = "SYSTEM"
		default:
			label = strings.ToUpper(string(m.Role))
		}
		fmt.Fprintf(&b, "%3d. %-6s (%d blocks)\n", i+1, label, len(m.Blocks))
	}
	sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
	return nil
}
