package ui

import (
	"fmt"
	"strings"

	"claudecode/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// PermissionModal asks the human whether a tool may run. Up/Down or j/k
// navigates the three options; Enter selects. The chosen response is sent
// into Reply before the modal dismisses, so the chat-side waiter unblocks.
type PermissionModal struct {
	Tool      string
	InputJSON string
	Reply     chan core.PermissionResponse

	cursor int
}

const (
	permAllowOnce = iota
	permDeny
	permAllowAlways
)

var permissionLabels = []string{
	"Allow once",
	"Deny",
	"Always allow this tool",
}

func (m *PermissionModal) Init() tea.Cmd { return nil }

func (m *PermissionModal) Title() string {
	return fmt.Sprintf("Permission: %s", m.Tool)
}

func (m *PermissionModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.Type {
	case tea.KeyEnter:
		m.send(m.responseFor(m.cursor))
		return nil, nil
	case tea.KeyEsc:
		// Esc denies, matching the safer default.
		m.send(core.PermissionResponse{Decision: core.PermissionDeny})
		return nil, nil
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case tea.KeyDown:
		if m.cursor < len(permissionLabels)-1 {
			m.cursor++
		}
		return m, nil
	case tea.KeyRunes:
		if len(key.Runes) == 0 {
			return m, nil
		}
		switch string(key.Runes) {
		case "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "j":
			if m.cursor < len(permissionLabels)-1 {
				m.cursor++
			}
		case "y":
			m.send(core.PermissionResponse{Decision: core.PermissionAllow})
			return nil, nil
		case "a":
			m.send(core.PermissionResponse{Decision: core.PermissionAllow, Remember: true})
			return nil, nil
		case "n", "d":
			m.send(core.PermissionResponse{Decision: core.PermissionDeny})
			return nil, nil
		}
	}
	return m, nil
}

// responseFor maps the highlighted option to a core.PermissionResponse.
// "Always allow" sets Remember=true so the gate persists the rule for
// the rest of the session.
func (m *PermissionModal) responseFor(idx int) core.PermissionResponse {
	switch idx {
	case permAllowOnce:
		return core.PermissionResponse{Decision: core.PermissionAllow}
	case permAllowAlways:
		return core.PermissionResponse{Decision: core.PermissionAllow, Remember: true}
	default:
		return core.PermissionResponse{Decision: core.PermissionDeny}
	}
}

// send forwards the response to the chat layer. Wrapped in a recover so a
// closed channel cannot panic the program.
func (m *PermissionModal) send(r core.PermissionResponse) {
	if m.Reply == nil {
		return
	}
	defer func() { _ = recover() }()
	m.Reply <- r
}

func (m *PermissionModal) View(width, height int) string {
	w := width
	if w > 80 {
		w = 80
	}

	var b strings.Builder
	b.WriteString("Tool wants to run:\n")
	preview := previewInput(m.InputJSON)
	if preview == "" {
		preview = "(no arguments)"
	}
	b.WriteString(toolNameStyle.Render(m.Tool))
	b.WriteString("\n")
	b.WriteString(thinkingStyle.Render(preview))
	b.WriteString("\n\n")

	for i, label := range permissionLabels {
		if i == m.cursor {
			b.WriteString(typeaheadSelectedStyle.Render("> " + label))
		} else {
			b.WriteString(typeaheadItemStyle.Render("  " + label))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(thinkingStyle.Render("(j/k or Up/Down, Enter select, Esc deny)"))

	frame := renderModalFrame(m.Title(), strings.TrimRight(b.String(), "\n"), w)
	return centerModal(frame, width, height)
}
