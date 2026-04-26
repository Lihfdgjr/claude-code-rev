package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type titleCmd struct{}

// NewTitle returns a /title command that reads or sets the session title.
func NewTitle() core.Command { return &titleCmd{} }

func (titleCmd) Name() string     { return "title" }
func (titleCmd) Synopsis() string { return "Show or set the session title" }

func (titleCmd) Run(ctx context.Context, args string, sess core.Session) error {
	args = strings.TrimSpace(args)
	if args == "" {
		t := sess.Title()
		if t == "" {
			sess.Notify(core.NotifyInfo, "title: (unset)")
		} else {
			sess.Notify(core.NotifyInfo, fmt.Sprintf("title: %s", t))
		}
		return nil
	}
	sess.SetTitle(args)
	sess.Notify(core.NotifyInfo, fmt.Sprintf("title set: %s", args))
	return nil
}
