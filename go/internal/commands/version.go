package commands

import (
	"context"

	"claudecode/internal/core"
)

const versionString = "claudecode-go 0.1.0"

type versionCmd struct{}

func NewVersion() core.Command { return &versionCmd{} }

func (versionCmd) Name() string     { return "version" }
func (versionCmd) Synopsis() string { return "Print the CLI version" }

func (versionCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo, versionString)
	return nil
}
