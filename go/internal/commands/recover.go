package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
	"claudecode/internal/sessions"
)

type recoverCmd struct {
	store          *sessions.Store
	transcriptRoot string
}

// NewRecover returns a /recover command that lists recently active sessions
// eligible for resumption.
func NewRecover(store *sessions.Store, transcriptRoot string) core.Command {
	return &recoverCmd{store: store, transcriptRoot: transcriptRoot}
}

func (c *recoverCmd) Name() string { return "recover" }
func (c *recoverCmd) Synopsis() string {
	return "List recently active sessions eligible for crash recovery"
}

func (c *recoverCmd) Run(ctx context.Context, args string, sess core.Session) error {
	cands, err := c.store.Recover(c.transcriptRoot)
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("recover: %v", err))
		return err
	}
	if len(cands) == 0 {
		sess.Notify(core.NotifyInfo, "no recoverable sessions in the last 24h")
		return nil
	}
	var b strings.Builder
	b.WriteString("Recoverable sessions (use `/resume <id>` to restore):\n")
	for _, cand := range cands {
		tr := "no"
		if cand.HasTranscript {
			tr = "yes"
		}
		title := cand.Title
		if title == "" {
			title = "(untitled)"
		}
		fmt.Fprintf(&b, "  %s | %s | %d msgs | transcript=%s | %s\n",
			cand.ID,
			cand.Modified.Format("2006-01-02 15:04:05"),
			cand.MessageCount,
			tr,
			title,
		)
	}
	sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
	return nil
}
