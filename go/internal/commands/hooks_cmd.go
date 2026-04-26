package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"claudecode/internal/core"
	"claudecode/internal/hooks"
)

type hooksCmd struct {
	cfg hooks.Config
}

func NewHooks(cfg hooks.Config) core.Command { return &hooksCmd{cfg: cfg} }

func (hooksCmd) Name() string     { return "hooks" }
func (hooksCmd) Synopsis() string { return "List configured hooks" }

func (c hooksCmd) Run(ctx context.Context, args string, sess core.Session) error {
	if len(c.cfg) == 0 {
		sess.Notify(core.NotifyInfo, "No hooks configured. Edit ~/.claude/settings.json to add hooks under the \"hooks\" key.")
		return nil
	}

	names := make([]string, 0, len(c.cfg))
	for name := range c.cfg {
		names = append(names, string(name))
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString("Configured hooks:\n")
	for _, name := range names {
		specs := c.cfg[hooks.EventName(name)]
		fmt.Fprintf(&b, "  %s:\n", name)
		for _, s := range specs {
			matcher := s.Matcher
			if matcher == "" {
				matcher = "*"
			}
			timeout := s.Timeout
			if timeout <= 0 {
				timeout = 30
			}
			fmt.Fprintf(&b, "    - matcher=%s type=%s timeout=%ds command=%s\n", matcher, s.Type, timeout, s.Command)
		}
	}
	sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
	return nil
}
