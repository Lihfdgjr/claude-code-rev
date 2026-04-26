package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"claudecode/internal/core"
	"claudecode/internal/oauth"
	"claudecode/internal/telemetry"
)

type loginCmd struct {
	store *oauth.Store
}

func NewLogin(store *oauth.Store) core.Command {
	return &loginCmd{store: store}
}

func (c *loginCmd) Name() string     { return "login" }
func (c *loginCmd) Synopsis() string { return "Store an Anthropic API token or print login instructions" }

func (c *loginCmd) Run(ctx context.Context, args string, sess core.Session) error {
	arg := strings.TrimSpace(args)
	switch {
	case arg == "":
		sess.Notify(core.NotifyInfo,
			"Login options:\n"+
				"  1. Set ANTHROPIC_API_KEY in your environment.\n"+
				"  2. Run `/login <token>` to store a token directly.\n"+
				"  3. Run `/login --browser` to start a browser-based flow.\n"+
				"  4. Add `api_key` to ~/.claude/settings.json.")
		return nil
	case arg == "--browser":
		flow := oauth.NewFlow("", "", "")
		code, err := flow.Begin(ctx)
		if err != nil {
			sess.Notify(core.NotifyWarn, "Browser login unavailable: "+err.Error())
			c.logEvent("login.browser.failed", map[string]interface{}{"error": err.Error()})
			return nil
		}
		tok, err := flow.Exchange(ctx, code)
		if err != nil {
			return fmt.Errorf("oauth exchange: %w", err)
		}
		if err := c.store.Save(tok); err != nil {
			return fmt.Errorf("save token: %w", err)
		}
		sess.Notify(core.NotifyInfo, "Token stored at "+c.store.Path())
		c.logEvent("login.browser.ok", nil)
		return nil
	default:
		t := &oauth.Token{
			AccessToken: arg,
			TokenType:   "Bearer",
			ExpiresAt:   time.Now().Add(365 * 24 * time.Hour),
		}
		if err := c.store.Save(t); err != nil {
			return fmt.Errorf("save token: %w", err)
		}
		sess.Notify(core.NotifyInfo, "Token stored at "+c.store.Path())
		c.logEvent("login.token.ok", nil)
		return nil
	}
}

func (c *loginCmd) logEvent(kind string, data map[string]interface{}) {
	if l := telemetry.Global(); l != nil {
		_ = l.Log(telemetry.Event{Kind: kind, Data: data})
	}
}
