package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
	"claudecode/internal/ui"
)

type themeCmd struct{}

func NewTheme() core.Command { return &themeCmd{} }

func (themeCmd) Name() string     { return "theme" }
func (themeCmd) Synopsis() string { return "List or switch UI themes" }

func (themeCmd) Run(ctx context.Context, args string, sess core.Session) error {
	name := strings.TrimSpace(args)
	if name == "" {
		var b strings.Builder
		b.WriteString("Available themes:\n")
		active := ""
		if t := ui.ActiveTheme.Load(); t != nil {
			active = t.Name
		}
		for _, t := range ui.ListThemes() {
			marker := "  "
			if t.Name == active {
				marker = "* "
			}
			b.WriteString(fmt.Sprintf("%s%-14s %s\n", marker, t.Name, t.Description))
		}
		b.WriteString("\nUse /theme <name> to switch.")
		sess.Notify(core.NotifyInfo, b.String())
		return nil
	}
	if err := ui.ApplyTheme(name); err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("theme: %v", err))
		return nil
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Theme set to %q.", name))
	return nil
}
