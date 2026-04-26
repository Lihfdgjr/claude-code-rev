package commands

import (
	"context"
	"fmt"

	"claudecode/internal/core"
	"claudecode/internal/ide"
)

type ideCmd struct{}

// NewIDE returns the /ide command, which prints the detected host editor.
func NewIDE() core.Command { return &ideCmd{} }

func (ideCmd) Name() string     { return "ide" }
func (ideCmd) Synopsis() string { return "Show the detected host IDE" }

func (ideCmd) Run(ctx context.Context, args string, sess core.Session) error {
	host := ide.Detect()
	name := host.Name()
	if name == "" || name == "unknown" {
		sess.Notify(core.NotifyInfo, "IDE: unknown. Set TERM_PROGRAM, VSCODE_INJECTION, or JETBRAINS_IDE to identify your editor.")
		return nil
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("IDE: %s (integration is stubbed in this build)", name))
	return nil
}
