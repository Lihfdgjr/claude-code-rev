package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Binding describes one user-visible keyboard shortcut. Action is optional;
// when nil the binding is documentation-only (handled inline in handleKey).
type Binding struct {
	Name        string
	Keys        string
	Description string
	Action      func(*Model) tea.Cmd
}

// Bindings is the registry of major shortcuts surfaced via /keybindings.
var Bindings = []Binding{
	{Name: "submit", Keys: "Enter", Description: "Submit input or run /command"},
	{Name: "newline", Keys: "Alt+Enter", Description: "Insert newline in input"},
	{Name: "cancel", Keys: "Esc", Description: "Clear input or dismiss modal"},
	{Name: "quit", Keys: "Ctrl+C", Description: "Cancel turn or quit"},
	{Name: "complete-next", Keys: "Tab", Description: "Cycle next typeahead suggestion"},
	{Name: "complete-prev", Keys: "Shift+Tab", Description: "Cycle previous typeahead suggestion"},
	{Name: "filepicker", Keys: "Ctrl+@", Description: "Open file picker"},
	{Name: "search", Keys: "Ctrl+F", Description: "Search history"},
	{Name: "settings", Keys: "Ctrl+,", Description: "Open settings editor"},
	{Name: "page-up", Keys: "PgUp", Description: "Scroll viewport up by half a page"},
	{Name: "page-down", Keys: "PgDn", Description: "Scroll viewport down by half a page"},
}

// RenderBindings produces a table of shortcuts suitable for /keybindings.
func RenderBindings(width int) string {
	maxKeys := 0
	maxName := 0
	for _, b := range Bindings {
		if n := len(b.Keys); n > maxKeys {
			maxKeys = n
		}
		if n := len(b.Name); n > maxName {
			maxName = n
		}
	}

	var b strings.Builder
	b.WriteString("Keyboard shortcuts:\n")
	for _, k := range Bindings {
		row := fmt.Sprintf("  %s  %s  %s",
			padRight(k.Keys, maxKeys),
			padRight(k.Name, maxName),
			k.Description,
		)
		if width > 0 && len(row) > width {
			row = row[:width]
		}
		b.WriteString(row)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
