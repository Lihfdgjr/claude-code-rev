package commands

import (
	"context"

	"claudecode/internal/core"
)

type editorCmd struct{}

func NewEditor() core.Command { return &editorCmd{} }

func (editorCmd) Name() string     { return "editor" }
func (editorCmd) Synopsis() string { return "Compose a long prompt in your $EDITOR" }

func (editorCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo, "Use $EDITOR to compose a long prompt; paste back into chat.")
	return nil
}
