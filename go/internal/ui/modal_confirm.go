package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ConfirmModal asks the user a yes/no question and reports the answer via
// OnDecide. Y or Enter confirms; N or Esc denies. The returned tea.Cmd from
// OnDecide is propagated upward.
type ConfirmModal struct {
	TitleText string
	Body      string
	OnDecide  func(bool) tea.Cmd
}

func (m *ConfirmModal) Init() tea.Cmd { return nil }

func (m *ConfirmModal) Title() string { return m.TitleText }

func (m *ConfirmModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.Type {
	case tea.KeyEnter:
		return nil, m.decide(true)
	case tea.KeyEsc:
		return nil, m.decide(false)
	case tea.KeyRunes:
		if len(key.Runes) == 0 {
			return m, nil
		}
		switch strings.ToLower(string(key.Runes)) {
		case "y":
			return nil, m.decide(true)
		case "n":
			return nil, m.decide(false)
		}
	}
	return m, nil
}

func (m *ConfirmModal) decide(ok bool) tea.Cmd {
	if m.OnDecide == nil {
		return nil
	}
	return m.OnDecide(ok)
}

func (m *ConfirmModal) View(width, height int) string {
	w := width
	if w > 70 {
		w = 70
	}
	hint := "[Y]es / [N]o   (Enter=yes, Esc=no)"
	body := m.Body + "\n\n" + hint
	frame := renderModalFrame(m.TitleText, body, w)
	return centerModal(frame, width, height)
}
