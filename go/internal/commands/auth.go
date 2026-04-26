package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claudecode/internal/core"
)

type authCmd struct{}

func NewAuth() core.Command { return &authCmd{} }

func (authCmd) Name() string     { return "auth" }
func (authCmd) Synopsis() string { return "Report API key and credentials status" }

func (authCmd) Run(ctx context.Context, args string, sess core.Session) error {
	var b strings.Builder
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		b.WriteString("ANTHROPIC_API_KEY: not set\n")
	} else {
		fmt.Fprintf(&b, "ANTHROPIC_API_KEY: %s\n", maskKey(key))
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(&b, "credentials.json: home dir error: %v", err)
		sess.Notify(core.NotifyInfo, b.String())
		return nil
	}
	credPath := filepath.Join(home, ".claude", "credentials.json")
	if _, err := os.Stat(credPath); err == nil {
		fmt.Fprintf(&b, "credentials.json: present at %s", credPath)
	} else {
		fmt.Fprintf(&b, "credentials.json: missing (%s)", credPath)
	}
	sess.Notify(core.NotifyInfo, b.String())
	return nil
}

func maskKey(s string) string {
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-8) + s[len(s)-4:]
}
