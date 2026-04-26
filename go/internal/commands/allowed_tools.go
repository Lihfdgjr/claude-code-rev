package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claudecode/internal/core"
)

type allowedToolsCmd struct{}

func NewAllowedTools() core.Command { return &allowedToolsCmd{} }

func (allowedToolsCmd) Name() string     { return "allowed-tools" }
func (allowedToolsCmd) Synopsis() string { return "List tools allowed by ~/.claude/settings.json" }

func (allowedToolsCmd) Run(ctx context.Context, args string, sess core.Session) error {
	home, err := os.UserHomeDir()
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("home dir: %v", err))
		return nil
	}
	path := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		sess.Notify(core.NotifyWarn, fmt.Sprintf("settings.json not readable at %s: %v", path, err))
		return nil
	}
	var settings struct {
		Permissions struct {
			Allow []string `json:"allow"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("parse settings.json: %v", err))
		return nil
	}
	if len(settings.Permissions.Allow) == 0 {
		sess.Notify(core.NotifyInfo, "(no allowed tools configured)")
		return nil
	}
	var b strings.Builder
	b.WriteString("Allowed entries:\n")
	for _, e := range settings.Permissions.Allow {
		fmt.Fprintf(&b, "  - %s\n", e)
	}
	sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
	return nil
}
