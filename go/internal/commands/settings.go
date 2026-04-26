package commands

import (
	"context"
	"os"
	"path/filepath"

	"claudecode/internal/core"
)

type settingsCmd struct{}

func NewSettings() core.Command { return &settingsCmd{} }

func (settingsCmd) Name() string     { return "settings" }
func (settingsCmd) Synopsis() string { return "Open the interactive settings editor" }

func (settingsCmd) Run(ctx context.Context, args string, sess core.Session) error {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".claude", "settings.json")
	sess.Notify(core.NotifyInfo,
		"Settings live in "+path+
			". Press Ctrl+, in the TUI to open the editor.")
	return nil
}
