package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type transcriptCmd struct{}

func NewTranscript() core.Command { return &transcriptCmd{} }

func (transcriptCmd) Name() string     { return "transcript" }
func (transcriptCmd) Synopsis() string { return "Print the full conversation transcript" }

func (transcriptCmd) Run(ctx context.Context, args string, sess core.Session) error {
	hist := sess.History()
	if len(hist) == 0 {
		sess.Notify(core.NotifyInfo, "(empty transcript)")
		return nil
	}
	var b strings.Builder
	for i, m := range hist {
		if i > 0 {
			b.WriteString("\n---\n")
		}
		fmt.Fprintf(&b, "[%s]\n", m.Role)
		for _, blk := range m.Blocks {
			switch v := blk.(type) {
			case core.TextBlock:
				b.WriteString(v.Text)
				b.WriteString("\n")
			case core.ThinkingBlock:
				fmt.Fprintf(&b, "(thinking) %s\n", v.Text)
			case core.ToolUseBlock:
				fmt.Fprintf(&b, "(tool_use %s id=%s) %s\n", v.Name, v.ID, string(v.Input))
			case core.ToolResultBlock:
				marker := "tool_result"
				if v.IsError {
					marker = "tool_error"
				}
				fmt.Fprintf(&b, "(%s use_id=%s)\n%s\n", marker, v.UseID, v.Content)
			}
		}
	}
	sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
	return nil
}
