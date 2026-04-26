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

type configCmd struct{}

func NewConfig() core.Command { return &configCmd{} }

func (configCmd) Name() string     { return "config" }
func (configCmd) Synopsis() string { return "Show or inspect ~/.claude/settings.json" }

func (configCmd) Run(ctx context.Context, args string, sess core.Session) error {
	home, err := os.UserHomeDir()
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("config: %v", err))
		return nil
	}
	path := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		sess.Notify(core.NotifyInfo, fmt.Sprintf("no settings.json found at %s", path))
		return nil
	}

	fields := strings.Fields(args)
	switch len(fields) {
	case 0:
		sess.Notify(core.NotifyInfo, string(data))
	case 1:
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			sess.Notify(core.NotifyError, fmt.Sprintf("config: %v", err))
			return nil
		}
		v, ok := m[fields[0]]
		if !ok {
			sess.Notify(core.NotifyInfo, fmt.Sprintf("%s: (unset)", fields[0]))
			return nil
		}
		out, _ := json.MarshalIndent(v, "", "  ")
		sess.Notify(core.NotifyInfo, fmt.Sprintf("%s = %s", fields[0], out))
	default:
		sess.Notify(core.NotifyInfo, fmt.Sprintf("To set %s, edit %s manually.", fields[0], path))
	}
	return nil
}
