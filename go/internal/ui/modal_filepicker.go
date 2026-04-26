package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// FilePickerModal lets the user pick a file (or directory) starting at Cwd.
// Typing acts as an incremental filter on the visible entries; arrow keys
// move the cursor; Enter selects the highlighted item; Esc cancels.
type FilePickerModal struct {
	Cwd      string
	Items    []os.DirEntry
	Cursor   int
	OnSelect func(path string) tea.Cmd

	filter string
	offset int
}

// NewFilePickerModal constructs a modal initialised by listing dir.
func NewFilePickerModal(dir string, onSelect func(path string) tea.Cmd) *FilePickerModal {
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	entries := readDirSorted(abs)
	return &FilePickerModal{
		Cwd:      abs,
		Items:    entries,
		OnSelect: onSelect,
	}
}

func readDirSorted(dir string) []os.DirEntry {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	sort.SliceStable(ents, func(i, j int) bool {
		// Directories first, then alpha.
		if ents[i].IsDir() != ents[j].IsDir() {
			return ents[i].IsDir()
		}
		return strings.ToLower(ents[i].Name()) < strings.ToLower(ents[j].Name())
	})
	return ents
}

func (m *FilePickerModal) Init() tea.Cmd { return nil }

func (m *FilePickerModal) Title() string {
	return "Pick file: " + m.Cwd
}

const filePickerVisibleRows = 10

func (m *FilePickerModal) filtered() []os.DirEntry {
	if m.filter == "" {
		return m.Items
	}
	prefix := strings.ToLower(m.filter)
	out := make([]os.DirEntry, 0, len(m.Items))
	for _, e := range m.Items {
		if strings.HasPrefix(strings.ToLower(e.Name()), prefix) {
			out = append(out, e)
		}
	}
	return out
}

func (m *FilePickerModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	items := m.filtered()

	switch key.Type {
	case tea.KeyEnter:
		if m.OnSelect == nil || len(items) == 0 {
			return nil, nil
		}
		sel := items[m.clampCursor(items)]
		full := filepath.Join(m.Cwd, sel.Name())
		if sel.IsDir() {
			// Descend into the directory rather than selecting it.
			m.Cwd = full
			m.Items = readDirSorted(full)
			m.filter = ""
			m.Cursor = 0
			m.offset = 0
			return m, nil
		}
		return nil, m.OnSelect(full)

	case tea.KeyEsc:
		return nil, nil

	case tea.KeyUp:
		if m.Cursor > 0 {
			m.Cursor--
		}
		if m.Cursor < m.offset {
			m.offset = m.Cursor
		}
		return m, nil

	case tea.KeyDown:
		if m.Cursor < len(items)-1 {
			m.Cursor++
		}
		if m.Cursor >= m.offset+filePickerVisibleRows {
			m.offset = m.Cursor - filePickerVisibleRows + 1
		}
		return m, nil

	case tea.KeyBackspace:
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.Cursor = 0
			m.offset = 0
		} else if parent := filepath.Dir(m.Cwd); parent != m.Cwd {
			m.Cwd = parent
			m.Items = readDirSorted(parent)
			m.Cursor = 0
			m.offset = 0
		}
		return m, nil

	case tea.KeyRunes:
		if len(key.Runes) == 0 {
			return m, nil
		}
		m.filter += string(key.Runes)
		m.Cursor = 0
		m.offset = 0
		return m, nil

	case tea.KeySpace:
		m.filter += " "
		return m, nil
	}
	return m, nil
}

func (m *FilePickerModal) clampCursor(items []os.DirEntry) int {
	if m.Cursor < 0 {
		m.Cursor = 0
	}
	if m.Cursor >= len(items) {
		m.Cursor = len(items) - 1
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
	return m.Cursor
}

func (m *FilePickerModal) View(width, height int) string {
	w := width
	if w > 80 {
		w = 80
	}

	items := m.filtered()
	if len(items) == 0 {
		body := "(no entries"
		if m.filter != "" {
			body += " match \"" + m.filter + "\""
		}
		body += ")\n\n" + thinkingStyle.Render("Esc cancel · Backspace clear/up dir")
		return centerModal(renderModalFrame(m.Title(), body, w), width, height)
	}

	cursor := m.clampCursor(items)
	if cursor >= m.offset+filePickerVisibleRows {
		m.offset = cursor - filePickerVisibleRows + 1
	}
	if cursor < m.offset {
		m.offset = cursor
	}

	end := m.offset + filePickerVisibleRows
	if end > len(items) {
		end = len(items)
	}

	var b strings.Builder
	for i := m.offset; i < end; i++ {
		name := items[i].Name()
		if items[i].IsDir() {
			name += "/"
		}
		if i == cursor {
			b.WriteString(typeaheadSelectedStyle.Render("> " + name))
		} else {
			b.WriteString(typeaheadItemStyle.Render("  " + name))
		}
		b.WriteString("\n")
	}

	footer := fmt.Sprintf("\nfilter: %q   %d/%d", m.filter, cursor+1, len(items))
	footer += "\n" + "(type to filter, Enter select/descend, Backspace up, Esc cancel)"
	b.WriteString(thinkingStyle.Render(footer))

	frame := renderModalFrame(m.Title(), strings.TrimRight(b.String(), "\n"), w)
	return centerModal(frame, width, height)
}
