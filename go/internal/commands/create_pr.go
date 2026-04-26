package commands

import (
	"context"

	"claudecode/internal/core"
)

type createPRCmd struct{}

func NewCreatePR() core.Command { return &createPRCmd{} }

func (createPRCmd) Name() string     { return "create-pr" }
func (createPRCmd) Synopsis() string { return "Suggest a gh pr create command" }

func (createPRCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo, "Use the Bash tool to run: gh pr create --title '...' --body '...'")
	return nil
}
