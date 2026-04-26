package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Modal is the small interface every overlay implements. Returning a nil
// Modal from Update signals dismissal so the host can pop the stack.
type Modal interface {
	Init() tea.Cmd
	Update(tea.Msg) (Modal, tea.Cmd)
	View(width, height int) string
	Title() string
}

// centerModal centers a rendered modal block inside the given viewport. It
// uses lipgloss.Place with center alignment on both axes.
func centerModal(content string, width, height int) string {
	if width <= 0 || height <= 0 {
		return content
	}
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderModalFrame wraps body with a titled border. Used by the concrete
// modals so they share a consistent look.
func renderModalFrame(title, body string, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 60
	}
	contentWidth := maxWidth - 4 // border + padding budget
	if contentWidth < 10 {
		contentWidth = 10
	}

	t := modalTitleStyle.Render(title)
	wrapped := softWrap(body, contentWidth)
	inner := strings.Join([]string{t, wrapped}, "\n")
	return modalBorderStyle.Width(contentWidth + 2).Render(inner)
}
