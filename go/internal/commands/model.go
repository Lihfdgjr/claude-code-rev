package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type modelCmd struct{}

func NewModel() core.Command { return &modelCmd{} }

func (modelCmd) Name() string     { return "model" }
func (modelCmd) Synopsis() string { return "Show or set the active model" }

func (modelCmd) Run(ctx context.Context, args string, sess core.Session) error {
	id := strings.TrimSpace(args)
	if id == "" {
		current := sess.Model()
		if current == "" {
			sess.Notify(core.NotifyInfo, "No model is currently set.")
		} else {
			sess.Notify(core.NotifyInfo, fmt.Sprintf("Current model: %s", current))
		}
		return nil
	}
	sess.SetModel(id)
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Model set to %s.", id))
	return nil
}
