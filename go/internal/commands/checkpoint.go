package commands

import (
	"context"
	"strings"

	"claudecode/internal/core"
)

type checkpointCmd struct{}

func NewCheckpoint() core.Command { return &checkpointCmd{} }

func (checkpointCmd) Name() string     { return "checkpoint" }
func (checkpointCmd) Synopsis() string { return "Record a manual history checkpoint" }

func (checkpointCmd) Run(ctx context.Context, args string, sess core.Session) error {
	label := strings.TrimSpace(args)
	if label == "" {
		label = "manual checkpoint"
	}
	sess.Checkpoint(label)
	sess.Notify(core.NotifyInfo, "Checkpoint saved: "+label+".")
	return nil
}
