package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// AskUserModal collects a single line of free-form text from the user in
// response to a tool's UIAskUserEvent. Enter posts the typed value into
// Reply; Esc posts an empty string to signal cancellation.
type AskUserModal struct {
	Question string
	Reply    chan string

	input textinput.Model
	sent  bool
}

// NewAskUserModal builds the modal with a focused text input.
func NewAskUserModal(question string, reply chan string) *AskUserModal {
	ti := textinput.New()
	ti.Placeholder = "type your answer"
	ti.Prompt = "> "
	ti.Focus()
	ti.CharLimit = 0
	return &AskUserModal{
		Question: question,
		Reply:    reply,
		input:    ti,
	}
}

func (m *AskUserModal) Init() tea.Cmd { return textinput.Blink }

func (m *AskUserModal) Title() string { return "Question" }

func (m *AskUserModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyEnter:
			m.send(m.input.Value())
			return nil, nil
		case tea.KeyEsc:
			m.send("")
			return nil, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *AskUserModal) send(s string) {
	if m.Reply == nil || m.sent {
		return
	}
	m.sent = true
	defer func() { _ = recover() }()
	m.Reply <- s
}

func (m *AskUserModal) View(width, height int) string {
	w := width
	if w > 80 {
		w = 80
	}

	var b strings.Builder
	b.WriteString(m.Question)
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	b.WriteString(thinkingStyle.Render("(Enter submit, Esc cancel)"))

	frame := renderModalFrame(m.Title(), strings.TrimRight(b.String(), "\n"), w)
	return centerModal(frame, width, height)
}
