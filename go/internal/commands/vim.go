package commands

import (
	"context"
	"strings"

	"claudecode/internal/core"
	"claudecode/internal/ui"
)

type vimCmd struct{}

func NewVim() core.Command { return &vimCmd{} }

func (vimCmd) Name() string     { return "vim" }
func (vimCmd) Synopsis() string { return "Toggle vim mode (/vim, /vim normal, /vim insert)" }

func (vimCmd) Run(ctx context.Context, args string, sess core.Session) error {
	arg := strings.ToLower(strings.TrimSpace(args))

	switch arg {
	case "":
		// Toggle on/off; ToggleVim already resets to insert on enable.
		if ui.ToggleVim() {
			sess.Notify(core.NotifyInfo, "Vim mode enabled (insert). Esc → normal, i/a → insert.")
		} else {
			sess.Notify(core.NotifyInfo, "Vim mode disabled.")
		}
		return nil

	case "normal", "n":
		if !ui.VimEnabled.Load() {
			sess.Notify(core.NotifyWarn, "Vim mode is off. Run /vim to enable it first.")
			return nil
		}
		ui.VimMode.Store(ui.VimModeNormal)
		sess.Notify(core.NotifyInfo, "Vim: normal mode.")
		return nil

	case "insert", "i":
		if !ui.VimEnabled.Load() {
			sess.Notify(core.NotifyWarn, "Vim mode is off. Run /vim to enable it first.")
			return nil
		}
		ui.VimMode.Store(ui.VimModeInsert)
		sess.Notify(core.NotifyInfo, "Vim: insert mode.")
		return nil

	default:
		sess.Notify(core.NotifyWarn, "Usage: /vim [normal|insert]")
		return nil
	}
}
