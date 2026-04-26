package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"claudecode/internal/core"
)

type statusCmd struct{}

func NewStatus() core.Command { return &statusCmd{} }

func (statusCmd) Name() string     { return "status" }
func (statusCmd) Synopsis() string { return "Show session status" }

func (statusCmd) Run(ctx context.Context, args string, sess core.Session) error {
	wd, err := os.Getwd()
	if err != nil {
		wd = "?"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "model:    %s\n", sess.Model())
	fmt.Fprintf(&b, "messages: %d\n", len(sess.History()))
	fmt.Fprintf(&b, "tokens:   (use /cost for usage)\n")
	fmt.Fprintf(&b, "cwd:      %s", wd)
	sess.Notify(core.NotifyInfo, b.String())
	return nil
}
