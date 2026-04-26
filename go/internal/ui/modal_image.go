package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ImageModal renders a previously-rendered ASCII preview inside a bordered
// frame. Esc/q dismisses; arrow keys / j-k scroll if the preview overflows.
type ImageModal struct {
	Path    string
	Preview string

	offset int
}

// NewImageModal builds a modal from an arbitrary path. If the file decodes
// as an image, Preview holds the ASCII rendering; otherwise it holds the
// decode error message so the user sees what went wrong without crashing.
func NewImageModal(path string, maxWidth int) *ImageModal {
	preview, err := renderImagePreview(path, maxWidth)
	if err != nil {
		preview = "(unable to decode " + path + ": " + err.Error() + ")"
	}
	return &ImageModal{Path: path, Preview: preview}
}

func (m *ImageModal) Init() tea.Cmd { return nil }

func (m *ImageModal) Title() string {
	return "Image: " + m.Path
}

const imageModalVisibleRows = 24

func (m *ImageModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.Type {
	case tea.KeyEsc:
		return nil, nil
	case tea.KeyUp:
		m.scroll(-1)
	case tea.KeyDown:
		m.scroll(1)
	case tea.KeyPgUp:
		m.scroll(-imageModalVisibleRows)
	case tea.KeyPgDown:
		m.scroll(imageModalVisibleRows)
	case tea.KeyRunes:
		switch string(key.Runes) {
		case "q":
			return nil, nil
		case "k":
			m.scroll(-1)
		case "j":
			m.scroll(1)
		}
	}
	return m, nil
}

func (m *ImageModal) scroll(delta int) {
	m.offset += delta
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m *ImageModal) View(width, height int) string {
	w := width
	if w > 100 {
		w = 100
	}

	lines := strings.Split(m.Preview, "\n")
	maxOff := len(lines) - imageModalVisibleRows
	if maxOff < 0 {
		maxOff = 0
	}
	if m.offset > maxOff {
		m.offset = maxOff
	}
	end := m.offset + imageModalVisibleRows
	if end > len(lines) {
		end = len(lines)
	}
	visible := strings.Join(lines[m.offset:end], "\n")

	hint := "(Esc/q close, j/k or PgUp/PgDn scroll)"
	body := visible + "\n\n" + thinkingStyle.Render(hint)

	frame := renderModalFrame(m.Title(), body, w)
	return centerModal(frame, width, height)
}
