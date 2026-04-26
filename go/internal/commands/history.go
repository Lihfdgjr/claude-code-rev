package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
	"claudecode/internal/sessions"
)

type historyCmd struct {
	store *sessions.Store
}

// NewHistory returns a /history command bound to the given store.
func NewHistory(store *sessions.Store) core.Command {
	return &historyCmd{store: store}
}

func (c *historyCmd) Name() string     { return "history" }
func (c *historyCmd) Synopsis() string { return "List the 20 most recent saved sessions" }

func (c *historyCmd) Run(ctx context.Context, args string, sess core.Session) error {
	metas, err := c.store.List()
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("history: %v", err))
		return err
	}
	if len(metas) == 0 {
		sess.Notify(core.NotifyInfo, "no saved sessions")
		return nil
	}
	limit := 20
	if len(metas) < limit {
		limit = len(metas)
	}
	var b strings.Builder
	for i := 0; i < limit; i++ {
		m := metas[i]
		fmt.Fprintf(&b, "%s | %s | %d msgs | %s\n",
			m.ID,
			m.LastModified.Format("2006-01-02 15:04:05"),
			m.MessageCount,
			m.Summary,
		)
	}
	sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
	return nil
}
