package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// TextModal shows a scrollable block of read-only text. Esc or q dismisses;
// PgUp/PgDn scroll a page, j/k scroll a line.
type TextModal struct {
	TitleText string
	Body      string

	offset int
}

func (m *TextModal) Init() tea.Cmd { return nil }

func (m *TextModal) Title() string { return m.TitleText }

const textModalVisibleRows = 16

func (m *TextModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.Type {
	case tea.KeyEsc:
		return nil, nil
	case tea.KeyPgUp:
		m.scroll(-textModalVisibleRows)
		return m, nil
	case tea.KeyPgDown:
		m.scroll(textModalVisibleRows)
		return m, nil
	case tea.KeyUp:
		m.scroll(-1)
		return m, nil
	case tea.KeyDown:
		m.scroll(1)
		return m, nil
	case tea.KeyRunes:
		if len(key.Runes) == 0 {
			return m, nil
		}
		switch string(key.Runes) {
		case "q":
			return nil, nil
		case "k":
			m.scroll(-1)
		case "j":
			m.scroll(1)
		case " ":
			m.scroll(textModalVisibleRows)
		}
	}
	return m, nil
}

func (m *TextModal) scroll(delta int) {
	m.offset += delta
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m *TextModal) View(width, height int) string {
	w := width
	if w > 90 {
		w = 90
	}

	lines := strings.Split(m.Body, "\n")
	maxOffset := len(lines) - textModalVisibleRows
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.offset > maxOffset {
		m.offset = maxOffset
	}

	end := m.offset + textModalVisibleRows
	if end > len(lines) {
		end = len(lines)
	}
	visible := strings.Join(lines[m.offset:end], "\n")

	hint := "(Esc/q close, PgUp/PgDn scroll)"
	if len(lines) > textModalVisibleRows {
		hint = "(Esc/q close, j/k or PgUp/PgDn scroll)"
	}
	body := visible + "\n\n" + hint

	frame := renderModalFrame(m.TitleText, body, w)
	return centerModal(frame, width, height)
}
