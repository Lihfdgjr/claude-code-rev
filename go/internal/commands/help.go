package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type helpCmd struct {
	reg core.CommandRegistry
}

func NewHelp(reg core.CommandRegistry) core.Command {
	return &helpCmd{reg: reg}
}

func (h *helpCmd) Name() string     { return "help" }
func (h *helpCmd) Synopsis() string { return "List available commands" }

func (h *helpCmd) Run(ctx context.Context, args string, sess core.Session) error {
	cmds := h.reg.All()
	width := 0
	for _, c := range cmds {
		if n := len(c.Name()); n > width {
			width = n
		}
	}
	var b strings.Builder
	b.WriteString("Available commands:\n")
	for _, c := range cmds {
		fmt.Fprintf(&b, "  /%-*s  %s\n", width, c.Name(), c.Synopsis())
	}
	sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
	return nil
}
