package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"claudecode/internal/core"
	"claudecode/internal/sessions"
)

type saveCmd struct {
	store *sessions.Store
}

// NewSave returns a /save command bound to the given session store.
func NewSave(store *sessions.Store) core.Command {
	return &saveCmd{store: store}
}

func (c *saveCmd) Name() string     { return "save" }
func (c *saveCmd) Synopsis() string { return "Save the current session under <name> (or title/timestamp)" }

func (c *saveCmd) Run(ctx context.Context, args string, sess core.Session) error {
	id := strings.TrimSpace(args)
	if id == "" {
		if t := strings.TrimSpace(sess.Title()); t != "" {
			id = slugify(t)
		}
	}
	if id == "" {
		id = "session-" + time.Now().UTC().Format("20060102-150405")
	}

	snap := sessions.SnapshotFromSession(id, sess)
	if err := c.store.Save(id, snap); err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("save: %v", err))
		return err
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("saved %s.json", id))
	return nil
}

func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r == '-' || r == '_':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-_")
	if out == "" {
		return ""
	}
	if len(out) > 60 {
		out = out[:60]
	}
	return out
}
