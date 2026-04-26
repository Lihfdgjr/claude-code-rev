package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type watchCmd struct{}

func NewWatch() core.Command { return &watchCmd{} }

func (watchCmd) Name() string     { return "watch" }
func (watchCmd) Synopsis() string { return "Suggest a file-tail command" }

func (watchCmd) Run(ctx context.Context, args string, sess core.Session) error {
	path := strings.TrimSpace(args)
	if path == "" {
		path = "<path>"
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Use the Bash tool with: tail -f %s\n(or PowerShell: Get-Content -Wait %s)", path, path))
	return nil
}
