package ui

import (
	"fmt"
	"strings"

	"claudecode/internal/core"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// SearchHit references a matching message in the conversation snapshot.
type SearchHit struct {
	MessageIdx int
	Snippet    string
}

// SearchOverlay implements Modal. It accepts a query, live-filters the
// snapshot for matching TextBlock content, and lets the user jump the host
// viewport to any hit.
type SearchOverlay struct {
	Query   string
	Results []SearchHit
	Cursor  int

	host  *Model
	input textinput.Model
}

// NewSearchOverlay constructs a search overlay bound to the host model so it
// can read the snapshot and mutate the viewport on Enter.
func NewSearchOverlay(host *Model) *SearchOverlay {
	ti := textinput.New()
	ti.Placeholder = "search history..."
	ti.Prompt = "/ "
	ti.Focus()
	ti.CharLimit = 0
	s := &SearchOverlay{host: host, input: ti}
	s.recompute()
	return s
}

func (s *SearchOverlay) Init() tea.Cmd { return textinput.Blink }

func (s *SearchOverlay) Title() string { return "Search" }

func (s *SearchOverlay) Update(msg tea.Msg) (Modal, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyEsc:
			return nil, nil
		case tea.KeyEnter:
			s.applyJump()
			return nil, nil
		case tea.KeyUp:
			s.move(-1)
			return s, nil
		case tea.KeyDown:
			s.move(1)
			return s, nil
		case tea.KeyCtrlN:
			s.move(1)
			return s, nil
		case tea.KeyCtrlP:
			s.move(-1)
			return s, nil
		}
	}
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	if s.input.Value() != s.Query {
		s.Query = s.input.Value()
		s.recompute()
	}
	return s, cmd
}

func (s *SearchOverlay) move(d int) {
	if len(s.Results) == 0 {
		s.Cursor = 0
		return
	}
	s.Cursor += d
	if s.Cursor < 0 {
		s.Cursor = 0
	}
	if s.Cursor >= len(s.Results) {
		s.Cursor = len(s.Results) - 1
	}
}

// recompute scans the snapshot for matches against Query. An empty query
// produces no results so the modal renders a hint instead of every message.
func (s *SearchOverlay) recompute() {
	s.Results = nil
	s.Cursor = 0
	q := strings.TrimSpace(s.Query)
	if q == "" || s.host == nil {
		return
	}
	hist := s.host.driver.Snapshot()
	needle := strings.ToLower(q)
	for i, m := range hist {
		for _, blk := range m.Blocks {
			tb, ok := blk.(core.TextBlock)
			if !ok {
				continue
			}
			lower := strings.ToLower(tb.Text)
			idx := strings.Index(lower, needle)
			if idx < 0 {
				continue
			}
			s.Results = append(s.Results, SearchHit{
				MessageIdx: i,
				Snippet:    snippetAround(tb.Text, idx, len(needle), 60),
			})
			break
		}
	}
}

// applyJump scrolls the host viewport so the selected message sits near the
// top. The line offset is approximated by re-rendering the snapshot prefix
// up to the chosen message and counting the resulting line count.
func (s *SearchOverlay) applyJump() {
	if s.host == nil || len(s.Results) == 0 {
		return
	}
	target := s.Results[s.Cursor].MessageIdx
	hist := s.host.driver.Snapshot()
	if target < 0 || target >= len(hist) {
		return
	}
	prefix := renderHistory(hist[:target], s.host.viewport.Width)
	off := 0
	if prefix != "" {
		off = strings.Count(prefix, "\n") + 1
	}
	s.host.viewport.SetYOffset(off)
}

// snippetAround returns up to ctx characters either side of idx with the
// match centered. Newlines are collapsed to spaces.
func snippetAround(text string, idx, mlen, ctx int) string {
	if idx < 0 {
		return strings.TrimSpace(collapseWhitespace(text))
	}
	start := idx - ctx
	if start < 0 {
		start = 0
	}
	end := idx + mlen + ctx
	if end > len(text) {
		end = len(text)
	}
	out := text[start:end]
	out = collapseWhitespace(out)
	if start > 0 {
		out = "..." + out
	}
	if end < len(text) {
		out = out + "..."
	}
	return out
}

func (s *SearchOverlay) View(width, height int) string {
	w := width
	if w > 90 {
		w = 90
	}

	var b strings.Builder
	b.WriteString(s.input.View())
	b.WriteString("\n")

	if strings.TrimSpace(s.Query) == "" {
		b.WriteString(thinkingStyle.Render("(type to search; Enter jumps, Esc closes)"))
	} else if len(s.Results) == 0 {
		b.WriteString(thinkingStyle.Render(fmt.Sprintf("no matches for %q", s.Query)))
	} else {
		const visible = 10
		start := 0
		if s.Cursor >= visible {
			start = s.Cursor - visible + 1
		}
		end := start + visible
		if end > len(s.Results) {
			end = len(s.Results)
		}
		for i := start; i < end; i++ {
			row := fmt.Sprintf("[#%d] %s", s.Results[i].MessageIdx, s.Results[i].Snippet)
			if i == s.Cursor {
				b.WriteString(typeaheadSelectedStyle.Render("> " + row))
			} else {
				b.WriteString(typeaheadItemStyle.Render("  " + row))
			}
			b.WriteString("\n")
		}
		b.WriteString(thinkingStyle.Render(fmt.Sprintf(
			"(%d/%d  Ctrl+N/P or Up/Down navigate, Enter jump, Esc close)",
			s.Cursor+1, len(s.Results))))
	}

	frame := renderModalFrame(s.Title(), strings.TrimRight(b.String(), "\n"), w)
	return centerModal(frame, width, height)
}
