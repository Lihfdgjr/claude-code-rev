package ui

import (
	"fmt"
	"strings"

	"claudecode/internal/core"
)

const citationCitedTextMax = 80

// renderCitations formats a numbered citation list. Returns an empty string
// when the slice is empty so callers can append unconditionally.
//
// Streaming note: TextDeltaEvent does not carry citations — they only arrive
// once the full message is committed to history. Bracketed `[N]` markers in
// streaming text are left untouched; the canonical render at end-of-block is
// where citations actually attach.
func renderCitations(cs []core.Citation, width int) string {
	if len(cs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(citationStyle.Render("Citations:"))
	for i, c := range cs {
		title := c.DocumentTitle
		if title == "" {
			title = "(untitled)"
		}
		cited := truncateCitation(c.CitedText, citationCitedTextMax)
		entry := fmt.Sprintf("[%d] %s: %q", i+1, title, cited)
		b.WriteString("\n")
		b.WriteString(citationStyle.Render(softWrap(entry, width)))
	}
	return b.String()
}

func truncateCitation(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len([]rune(s)) <= max {
		return s
	}
	rs := []rune(s)
	return string(rs[:max]) + "..."
}
