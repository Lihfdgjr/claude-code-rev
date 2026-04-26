package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"claudecode/internal/core"
)

type logsCmd struct{}

func NewLogs() core.Command { return &logsCmd{} }

func (logsCmd) Name() string     { return "logs" }
func (logsCmd) Synopsis() string { return "Print location of debug logs" }

func (logsCmd) Run(ctx context.Context, args string, sess core.Session) error {
	home, err := os.UserHomeDir()
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("home dir: %v", err))
		return nil
	}
	dir := filepath.Join(home, ".claude", "logs")
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Debug logs are written under: %s\nTail the most recent file to inspect activity.", dir))
	return nil
}
