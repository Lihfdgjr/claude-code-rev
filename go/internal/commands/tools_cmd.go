package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"claudecode/internal/core"
)

type toolsCmd struct{}

func NewTools() core.Command { return &toolsCmd{} }

func (toolsCmd) Name() string     { return "tools" }
func (toolsCmd) Synopsis() string { return "List tool names from CLAUDECODE_TOOLS or show hint" }

func (toolsCmd) Run(ctx context.Context, args string, sess core.Session) error {
	env := strings.TrimSpace(os.Getenv("CLAUDECODE_TOOLS"))
	if env == "" {
		sess.Notify(core.NotifyInfo, "Tool list is bound at startup. Set CLAUDECODE_TOOLS to override, or ask the model which tools are available.")
		return nil
	}
	parts := strings.Split(env, ",")
	var b strings.Builder
	b.WriteString("Tools (CLAUDECODE_TOOLS):\n")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		fmt.Fprintf(&b, "  - %s\n", p)
	}
	sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
	return nil
}
