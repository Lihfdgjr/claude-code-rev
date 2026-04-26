package commands

import (
	"context"

	"claudecode/internal/core"
	"claudecode/internal/ui"
)

type keybindingsCmd struct{}

func NewKeybindings() core.Command { return &keybindingsCmd{} }

func (keybindingsCmd) Name() string     { return "keybindings" }
func (keybindingsCmd) Synopsis() string { return "Show keyboard shortcuts" }

func (keybindingsCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo, ui.RenderBindings(80))
	return nil
}
