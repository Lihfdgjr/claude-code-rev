package ui

import (
	"strings"
	"sync/atomic"
	"time"

	"claudecode/internal/core"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LastError holds the most recent error reported by the network/retry layer.
// The chat package may write a non-nil error here to flip the connection
// status icon to red. Stored as atomic.Value because it's read on every
// View() refresh from the UI loop while writers run on background turns.
var LastError atomic.Value // holds error

const (
	toastTTL         = 4 * time.Second
	toastMaxVisible  = 3
	toastMaxWidth    = 36
	toastTickPeriod  = 500 * time.Millisecond
)

// Toast is a transient notification rendered as a small bordered box in the
// top-right corner. It expires automatically after ExpiresAt elapses.
type Toast struct {
	Text      string
	Level     core.NotifyLevel
	ExpiresAt time.Time
}

// toastTickMsg is a periodic prune wakeup.
type toastTickMsg struct{}

// scheduleToastTick returns a tea.Cmd that fires a toastTickMsg after a short
// delay so the model can prune expired toasts without busy-waiting.
func scheduleToastTick() tea.Cmd {
	return tea.Tick(toastTickPeriod, func(time.Time) tea.Msg { return toastTickMsg{} })
}

// addToast pushes a new toast onto the model with the standard TTL. If a
// duplicate of the most recent toast is added we just refresh its expiry
// instead of stacking dupes.
func (m *Model) addToast(level core.NotifyLevel, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	exp := time.Now().Add(toastTTL)
	if n := len(m.toasts); n > 0 {
		last := &m.toasts[n-1]
		if last.Text == text && last.Level == level {
			last.ExpiresAt = exp
			return
		}
	}
	m.toasts = append(m.toasts, Toast{Text: text, Level: level, ExpiresAt: exp})
	if len(m.toasts) > 16 {
		m.toasts = m.toasts[len(m.toasts)-16:]
	}
}

// pruneToasts drops expired entries. Returns true if any were removed.
func (m *Model) pruneToasts() bool {
	if len(m.toasts) == 0 {
		return false
	}
	now := time.Now()
	keep := m.toasts[:0]
	removed := false
	for _, t := range m.toasts {
		if now.Before(t.ExpiresAt) {
			keep = append(keep, t)
		} else {
			removed = true
		}
	}
	m.toasts = keep
	return removed
}

// activeToasts returns the most recent toastMaxVisible toasts.
func (m *Model) activeToasts() []Toast {
	if len(m.toasts) <= toastMaxVisible {
		return m.toasts
	}
	return m.toasts[len(m.toasts)-toastMaxVisible:]
}

// renderToasts composes the stacked toast column. When the model has no
// active toasts an empty string is returned.
func renderToasts(toasts []Toast) string {
	if len(toasts) == 0 {
		return ""
	}
	var b strings.Builder
	for i, t := range toasts {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(renderToastBox(t))
	}
	return b.String()
}

func renderToastBox(t Toast) string {
	style := toastInfoStyle
	icon := "i"
	switch t.Level {
	case core.NotifyError:
		style = toastErrStyle
		icon = "!"
	case core.NotifyWarn:
		style = toastWarnStyle
		icon = "!"
	case core.NotifyDebug:
		style = toastDebugStyle
		icon = "?"
	}
	body := truncateRunes(t.Text, toastMaxWidth-7)
	inner := icon + "  " + body
	return style.Render(inner)
}

// truncateRunes shortens s to at most n runes, appending "..." when clipped.
// The empty string and small n values are passed through unchanged.
func truncateRunes(s string, n int) string {
	if n <= 0 {
		return s
	}
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return string(rs[:n]) + "..."
}

// overlayToasts paints the toast column at the top-right of the base view by
// substituting characters on the affected rows. It works on rendered ANSI
// strings by splitting on "\n" and rewriting only the prefix-free trailing
// columns. To stay simple and ANSI-safe we render the toast column with its
// own padding and append it as a side panel on the first few rows; if the
// base content on a row is shorter than width-toast width we right-pad and
// concatenate, otherwise the toast clips to its own line below the base.
//
// In practice the simplest correct rendering is to append the toast block
// after the base view separated by a newline so it sits at the top of the
// next paint frame; Bubble Tea full-screen alt-screen mode will redraw the
// full buffer each tick. We instead overlay by replacing the right-most N
// columns of the first len(toasts) rows in `base`.
func overlayToasts(base string, toasts []Toast, width int) string {
	if len(toasts) == 0 || width <= 0 {
		return base
	}

	rendered := renderToasts(toasts)
	if rendered == "" {
		return base
	}

	tw := lipgloss.Width(rendered)
	if tw <= 0 || tw >= width {
		return base
	}

	// Split base into rows and toast into rows; for each toast row, right-align
	// it inside the corresponding base row by padding the base row up to
	// width-tw and appending the toast row. Rows beyond the toast height are
	// left untouched.
	baseRows := strings.Split(base, "\n")
	toastRows := strings.Split(rendered, "\n")

	for i, tr := range toastRows {
		if i >= len(baseRows) {
			break
		}
		row := baseRows[i]
		bw := lipgloss.Width(row)
		gap := width - tw - bw
		if gap < 1 {
			// Base row already too wide; truncate is risky (ANSI), so just
			// place the toast on its own line by appending after this row.
			row = row + "\n" + strings.Repeat(" ", width-tw) + tr
		} else {
			row = row + strings.Repeat(" ", gap) + tr
		}
		baseRows[i] = row
	}
	return strings.Join(baseRows, "\n")
}
