package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
	"claudecode/internal/sessions"
)

type resumeCmd struct {
	store *sessions.Store
}

// NewResume returns a /resume command bound to the given store.
func NewResume(store *sessions.Store) core.Command {
	return &resumeCmd{store: store}
}

func (c *resumeCmd) Name() string     { return "resume" }
func (c *resumeCmd) Synopsis() string { return "Resume a previous session by id, or list recent sessions" }

func (c *resumeCmd) Run(ctx context.Context, args string, sess core.Session) error {
	id := strings.TrimSpace(args)
	if id == "" {
		metas, err := c.store.List()
		if err != nil {
			sess.Notify(core.NotifyError, fmt.Sprintf("resume: %v", err))
			return err
		}
		if len(metas) == 0 {
			sess.Notify(core.NotifyInfo, "no saved sessions")
			return nil
		}
		limit := 10
		if len(metas) < limit {
			limit = len(metas)
		}
		var b strings.Builder
		b.WriteString("Recent sessions (re-run `/resume <id>`):\n")
		for i := 0; i < limit; i++ {
			m := metas[i]
			fmt.Fprintf(&b, "  %s | %s | %d msgs | %s\n",
				m.ID,
				m.LastModified.Format("2006-01-02 15:04:05"),
				m.MessageCount,
				m.Summary,
			)
		}
		sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
		return nil
	}

	snap, err := c.store.Load(id)
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("resume: %v", err))
		return err
	}
	msgs, err := sessions.DeserializeMessages(snap.Messages)
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("resume: %v", err))
		return err
	}

	sess.ResetHistory()
	for _, m := range msgs {
		sess.Append(m)
	}
	if snap.Model != "" {
		sess.SetModel(snap.Model)
	}
	if snap.SystemPrompt != "" {
		sess.SetSystemPrompt(snap.SystemPrompt)
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("resumed %s (%d messages)", snap.ID, len(msgs)))
	return nil
}
