package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// multilineBuf accumulates the lines committed via Alt+Enter / Shift+Enter
// while the live textinput holds the still-editable last line. The full
// input value is reconstructed at submit time as join(buf, "\n") + "\n" + input.
type multilineBuf struct {
	lines []string
}

// reset clears the multiline buffer.
func (b *multilineBuf) reset() {
	b.lines = nil
}

// commit appends the current input line to the multiline buffer and returns
// true if the buffer is now non-empty.
func (b *multilineBuf) commit(line string) {
	b.lines = append(b.lines, line)
}

// fullText returns the concatenated multiline value, joining buffered lines
// with "\n" and appending the live input line.
func (b *multilineBuf) fullText(live string) string {
	if len(b.lines) == 0 {
		return live
	}
	parts := make([]string, 0, len(b.lines)+1)
	parts = append(parts, b.lines...)
	parts = append(parts, live)
	return strings.Join(parts, "\n")
}

// nonEmpty reports whether the buffer holds any committed lines.
func (b *multilineBuf) nonEmpty() bool {
	return len(b.lines) > 0
}

// isMultilineNewlineKey reports whether a key event is the chord we hijack
// for "insert newline" instead of submitting. We accept Alt+Enter (the
// portable chord) and also treat any Enter that arrives with the Alt flag
// set as a request to break the line. Bubbletea on most terminals reports
// Shift+Enter as plain Enter so the user must use Alt+Enter; this is
// documented in the help banner.
func isMultilineNewlineKey(k tea.KeyMsg) bool {
	if k.Type != tea.KeyEnter {
		return false
	}
	return k.Alt
}

// renderMultilineInput composes the visible input area when the multiline
// buffer is non-empty. Buffered lines render above the live input row,
// indented to line up with the prompt. The first row carries the prompt
// itself, subsequent rows show a faint continuation marker.
func renderMultilineInput(buf *multilineBuf, prompt, liveView string, width int) string {
	if buf == nil || !buf.nonEmpty() {
		return renderInputLine(prompt, liveView, width)
	}

	pw := lipglossWidth(prompt)
	contIndent := ""
	if pw > 2 {
		contIndent = strings.Repeat(" ", pw-2) + thinkingStyle.Render("· ")
	} else {
		contIndent = thinkingStyle.Render("· ")
	}

	var b strings.Builder
	for i, ln := range buf.lines {
		if i == 0 {
			b.WriteString(inputPromptStyle.Render(prompt))
			b.WriteString(ln)
		} else {
			b.WriteString(contIndent)
			b.WriteString(ln)
		}
		b.WriteString("\n")
	}
	b.WriteString(contIndent)
	b.WriteString(liveView)
	_ = width
	return b.String()
}
