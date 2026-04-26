package commands

import (
	"context"
	"errors"
	"fmt"
	"os"

	"claudecode/internal/core"
	"claudecode/internal/oauth"
	"claudecode/internal/telemetry"
)

type logoutCmd struct {
	store *oauth.Store
}

func NewLogout(store *oauth.Store) core.Command {
	return &logoutCmd{store: store}
}

func (c *logoutCmd) Name() string     { return "logout" }
func (c *logoutCmd) Synopsis() string { return "Remove the stored Anthropic API token" }

func (c *logoutCmd) Run(ctx context.Context, args string, sess core.Session) error {
	if _, err := os.Stat(c.store.Path()); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			sess.Notify(core.NotifyInfo, "No token to clear.")
			return nil
		}
		return fmt.Errorf("stat token: %w", err)
	}
	if err := c.store.Clear(); err != nil {
		return fmt.Errorf("clear token: %w", err)
	}
	if l := telemetry.Global(); l != nil {
		_ = l.Log(telemetry.Event{Kind: "logout.ok"})
	}
	sess.Notify(core.NotifyInfo, "Token cleared.")
	return nil
}
