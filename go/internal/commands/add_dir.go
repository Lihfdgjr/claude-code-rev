package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type addDirCmd struct{}

func NewAddDir() core.Command { return &addDirCmd{} }

func (addDirCmd) Name() string     { return "add-dir" }
func (addDirCmd) Synopsis() string { return "Suggest adding a directory to allowed paths" }

func (addDirCmd) Run(ctx context.Context, args string, sess core.Session) error {
	path := strings.TrimSpace(args)
	if path == "" {
		sess.Notify(core.NotifyWarn, "Usage: /add-dir <path>")
		return nil
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Add to permissions.allow_dirs in ~/.claude/settings.json: %s", path))
	return nil
}
