package commands

import (
	"context"
	"fmt"
	"strings"

	"claudecode/internal/core"
	"claudecode/internal/sessions"
)

type findCmd struct {
	store *sessions.Store
}

// NewFind returns a /find command that searches saved session message content.
// Result quality scales with session count; this walks every session JSON.
func NewFind(store *sessions.Store) core.Command {
	return &findCmd{store: store}
}

func (c *findCmd) Name() string { return "find" }
func (c *findCmd) Synopsis() string {
	return "Find saved sessions whose message text contains <query>"
}

const (
	findMaxMatches  = 20
	findSnippetSpan = 50
)

func (c *findCmd) Run(ctx context.Context, args string, sess core.Session) error {
	query := strings.TrimSpace(args)
	if query == "" {
		sess.Notify(core.NotifyInfo, "usage: /find <query>")
		return nil
	}
	metas, err := c.store.List()
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("find: %v", err))
		return err
	}
	needle := strings.ToLower(query)
	var b strings.Builder
	matches := 0

	for _, m := range metas {
		if matches >= findMaxMatches {
			break
		}
		snap, err := c.store.Load(m.ID)
		if err != nil {
			continue
		}
		snippet := firstSnippet(snap, needle)
		if snippet == "" {
			continue
		}
		matches++
		fmt.Fprintf(&b, "%s | %s | %s\n",
			m.ID,
			m.LastModified.Format("2006-01-02 15:04:05"),
			snippet,
		)
	}

	if matches == 0 {
		sess.Notify(core.NotifyInfo, fmt.Sprintf("find: no matches for %q", query))
		return nil
	}
	sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
	return nil
}

// firstSnippet walks decoded message blocks and returns a 50-chars-around
// excerpt of the first text block that contains needle (case-insensitive).
func firstSnippet(snap *sessions.Snapshot, needle string) string {
	msgs, err := sessions.DeserializeMessages(snap.Messages)
	if err != nil {
		return ""
	}
	for _, m := range msgs {
		for _, blk := range m.Blocks {
			tb, ok := blk.(core.TextBlock)
			if !ok {
				continue
			}
			text := tb.Text
			lower := strings.ToLower(text)
			idx := strings.Index(lower, needle)
			if idx < 0 {
				continue
			}
			return excerpt(text, idx, len(needle))
		}
	}
	return ""
}

func excerpt(text string, idx, hitLen int) string {
	start := idx - findSnippetSpan
	if start < 0 {
		start = 0
	}
	end := idx + hitLen + findSnippetSpan
	if end > len(text) {
		end = len(text)
	}
	prefix := ""
	if start > 0 {
		prefix = "..."
	}
	suffix := ""
	if end < len(text) {
		suffix = "..."
	}
	out := prefix + text[start:end] + suffix
	out = strings.ReplaceAll(out, "\r", " ")
	out = strings.ReplaceAll(out, "\n", " ")
	return strings.TrimSpace(out)
}
