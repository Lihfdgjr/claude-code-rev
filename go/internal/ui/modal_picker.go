package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// PickerItem is one row of a PickerModal.
type PickerItem struct {
	Label  string
	Detail string
}

// PickerModal lets the user choose one of Items. Up/Down or k/j navigate,
// Enter selects, Esc cancels. The selected index is reported via OnSelect;
// a value of -1 means cancellation.
type PickerModal struct {
	TitleText string
	Items     []PickerItem
	OnSelect  func(int) tea.Cmd

	cursor int
	offset int // top-of-window index for scrolling
}

func (m *PickerModal) Init() tea.Cmd { return nil }

func (m *PickerModal) Title() string { return m.TitleText }

const pickerVisibleRows = 8

func (m *PickerModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.Type {
	case tea.KeyEnter:
		if m.OnSelect == nil || len(m.Items) == 0 {
			return nil, nil
		}
		return nil, m.OnSelect(m.cursor)
	case tea.KeyEsc:
		if m.OnSelect == nil {
			return nil, nil
		}
		return nil, m.OnSelect(-1)
	case tea.KeyUp:
		m.moveUp()
		return m, nil
	case tea.KeyDown:
		m.moveDown()
		return m, nil
	case tea.KeyRunes:
		if len(key.Runes) == 0 {
			return m, nil
		}
		switch string(key.Runes) {
		case "k":
			m.moveUp()
		case "j":
			m.moveDown()
		case "q":
			if m.OnSelect != nil {
				return nil, m.OnSelect(-1)
			}
			return nil, nil
		}
	}
	return m, nil
}

func (m *PickerModal) moveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
}

func (m *PickerModal) moveDown() {
	if m.cursor < len(m.Items)-1 {
		m.cursor++
	}
	if m.cursor >= m.offset+pickerVisibleRows {
		m.offset = m.cursor - pickerVisibleRows + 1
	}
}

func (m *PickerModal) View(width, height int) string {
	w := width
	if w > 80 {
		w = 80
	}

	if len(m.Items) == 0 {
		return centerModal(renderModalFrame(m.TitleText, "(no items)", w), width, height)
	}

	end := m.offset + pickerVisibleRows
	if end > len(m.Items) {
		end = len(m.Items)
	}

	var b strings.Builder
	for i := m.offset; i < end; i++ {
		row := m.Items[i].Label
		if m.Items[i].Detail != "" {
			row = fmt.Sprintf("%s  %s", row, m.Items[i].Detail)
		}
		if i == m.cursor {
			b.WriteString(typeaheadSelectedStyle.Render("> " + row))
		} else {
			b.WriteString(typeaheadItemStyle.Render("  " + row))
		}
		b.WriteString("\n")
	}
	if len(m.Items) > pickerVisibleRows {
		b.WriteString(thinkingStyle.Render(
			fmt.Sprintf("\n  %d/%d   (j/k navigate, Enter select, Esc cancel)",
				m.cursor+1, len(m.Items))))
	} else {
		b.WriteString(thinkingStyle.Render("\n  (j/k navigate, Enter select, Esc cancel)"))
	}

	frame := renderModalFrame(m.TitleText, strings.TrimRight(b.String(), "\n"), w)
	return centerModal(frame, width, height)
}
