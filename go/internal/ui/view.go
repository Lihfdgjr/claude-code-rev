package ui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"claudecode/internal/chat"
	"claudecode/internal/core"

	"github.com/charmbracelet/lipgloss"
)

// planActive reports whether plan mode is currently engaged. Wrapped so the
// cross-package read is testable from a single location.
func planActive() bool { return chat.PlanModeActive.Load() }

const (
	toolInputPreviewMax = 80
	toolResultMax       = 1000
)

// renderHistory turns the message history into a single string for the viewport.
func renderHistory(msgs []core.Message, width int) string {
	var b strings.Builder
	for i, m := range msgs {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(renderMessage(m, width))
	}
	return b.String()
}

func renderMessage(m core.Message, width int) string {
	var b strings.Builder
	switch m.Role {
	case core.RoleUser:
		b.WriteString(renderUserMessage(m, width))
	case core.RoleAssistant:
		b.WriteString(renderAssistantMessage(m, width))
	case core.RoleSystem:
		for _, blk := range m.Blocks {
			if t, ok := blk.(core.TextBlock); ok {
				b.WriteString(thinkingStyle.Render(t.Text))
				b.WriteString("\n")
			}
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderUserMessage(m core.Message, width int) string {
	var b strings.Builder
	for _, blk := range m.Blocks {
		switch v := blk.(type) {
		case core.TextBlock:
			lines := strings.Split(v.Text, "\n")
			for _, ln := range lines {
				b.WriteString(userPrefixStyle.Render("▌ "))
				b.WriteString(userTextStyle.Render(ln))
				b.WriteString("\n")
			}
		case core.ToolResultBlock:
			b.WriteString(renderToolResult(v.Content, v.IsError))
			b.WriteString("\n")
		}
	}
	return b.String()
}

func renderAssistantMessage(m core.Message, width int) string {
	var b strings.Builder
	for _, blk := range m.Blocks {
		switch v := blk.(type) {
		case core.TextBlock:
			text := renderMarkdown(v.Text)
			text = highlightFencedBlocks(text)
			b.WriteString(assistantStyle.Render(softWrap(text, width)))
			b.WriteString("\n")
			if cs := renderCitations(v.Citations, width); cs != "" {
				b.WriteString(cs)
				b.WriteString("\n")
			}
		case core.ThinkingBlock:
			b.WriteString(thinkingStyle.Render(softWrap("thinking: "+v.Text, width)))
			b.WriteString("\n")
		case core.ToolUseBlock:
			b.WriteString(renderToolStart(v.Name, string(v.Input)))
			b.WriteString("\n")
		case core.ToolResultBlock:
			b.WriteString(renderToolResult(v.Content, v.IsError))
			b.WriteString("\n")
		case core.ImageBlock:
			b.WriteString(renderAssistantImage(v, width))
			b.WriteString("\n")
		case core.AudioBlock:
			data, _ := base64.StdEncoding.DecodeString(v.Source)
			b.WriteString(thinkingStyle.Render(fmt.Sprintf("[audio: %s, %d bytes]", v.MediaType, len(data))))
			b.WriteString("\n")
		case core.DocumentBlock:
			data, _ := base64.StdEncoding.DecodeString(v.Source)
			title := v.Title
			if title == "" {
				title = "(untitled)"
			}
			b.WriteString(thinkingStyle.Render(fmt.Sprintf("[document: %s, %s, %d bytes]", title, v.MediaType, len(data))))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderAssistantImage decodes an assistant ImageBlock to a temp file and
// renders an ASCII shade preview. On any failure it falls back to a textual
// placeholder so the UI never breaks.
func renderAssistantImage(v core.ImageBlock, width int) string {
	data, err := base64.StdEncoding.DecodeString(v.Source)
	if err != nil || len(data) == 0 {
		return thinkingStyle.Render(fmt.Sprintf("[image: %s, %d bytes]", v.MediaType, len(data)))
	}
	f, err := os.CreateTemp("", "cc-img-*")
	if err != nil {
		return thinkingStyle.Render(fmt.Sprintf("[image: %s, %d bytes]", v.MediaType, len(data)))
	}
	tmpPath := f.Name()
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return thinkingStyle.Render(fmt.Sprintf("[image: %s, %d bytes]", v.MediaType, len(data)))
	}
	f.Close()
	defer os.Remove(tmpPath)

	maxW := width
	if maxW <= 0 {
		maxW = 64
	}
	preview, err := renderImagePreview(tmpPath, maxW)
	if err != nil {
		return thinkingStyle.Render(fmt.Sprintf("[image: %s, %d bytes]", v.MediaType, len(data)))
	}
	return preview
}

func renderToolStart(name, input string) string {
	preview := previewInput(input)
	header := toolStartStyle.Render("⚙ ") + toolNameStyle.Render(name)
	if preview != "" {
		header += toolStartStyle.Render("(" + preview + ")")
	} else {
		header += toolStartStyle.Render("()")
	}
	return header
}

func renderToolResult(content string, isError bool) string {
	trimmed := content
	if len(trimmed) > toolResultMax {
		trimmed = trimmed[:toolResultMax] + "..."
	}
	// Try diff rendering first; fall back to plain styling if not a diff.
	if !isError {
		if rendered, ok := renderDiffIfApplicable(trimmed, 0); ok {
			return toolResultStyle.Render(rendered)
		}
	}
	style := toolResultStyle
	if isError {
		style = style.Foreground(lipgloss.Color("9"))
	}
	return style.Render(trimmed)
}

func renderError(err error) string {
	if err == nil {
		return ""
	}
	return errorStyle.Render("error: " + err.Error())
}

// previewInput collapses raw JSON-ish tool input into a short single-line string.
func previewInput(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}
	// Try to parse as JSON object and produce a compact key=value preview.
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(s), &obj); err == nil {
		parts := make([]string, 0, len(obj))
		for k, v := range obj {
			parts = append(parts, fmt.Sprintf("%s=%s", k, shortValue(v)))
		}
		s = strings.Join(parts, ", ")
	} else {
		s = collapseWhitespace(s)
	}
	if len(s) > toolInputPreviewMax {
		s = s[:toolInputPreviewMax] + "..."
	}
	return s
}

func shortValue(v interface{}) string {
	switch t := v.(type) {
	case string:
		s := collapseWhitespace(t)
		if len(s) > 40 {
			s = s[:40] + "..."
		}
		return strconvQuote(s)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "?"
		}
		s := string(b)
		if len(s) > 40 {
			s = s[:40] + "..."
		}
		return s
	}
}

func strconvQuote(s string) string {
	return "\"" + s + "\""
}

func collapseWhitespace(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if r == '\n' || r == '\t' || r == '\r' {
			r = ' '
		}
		if r == ' ' {
			if prevSpace {
				continue
			}
			prevSpace = true
		} else {
			prevSpace = false
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// softWrap relies on lipgloss to perform width-aware soft wrapping. If width
// is zero we return the text unchanged.
func softWrap(text string, width int) string {
	if width <= 0 {
		return text
	}
	return lipgloss.NewStyle().Width(width).Render(text)
}

// renderStatusBar composes the bottom status bar.
func renderStatusBar(width int, model string, usage core.Usage, errMsg string) string {
	icons := renderStatusIcons()
	left := statusModelStyle.Render(model)
	usageStr := fmt.Sprintf(" in:%d out:%d", usage.InputTokens, usage.OutputTokens)
	if usage.CacheReadTokens > 0 || usage.CacheCreationTokens > 0 {
		usageStr += fmt.Sprintf(" cache:%d/%d", usage.CacheReadTokens, usage.CacheCreationTokens)
	}
	mid := statusUsageStyle.Render(usageStr)
	hint := statusHintStyle.Render("  ^C cancel/quit · / commands · ^F search")

	if errMsg != "" {
		hint = statusErrStyle.Render("  " + errMsg)
	}

	content := icons + left + mid + hint
	if width > 0 {
		// Pad to full width with status background.
		visibleLen := lipgloss.Width(content)
		if visibleLen < width {
			pad := strings.Repeat(" ", width-visibleLen)
			content += statusBarStyle.Render(pad)
		}
	}
	return content
}

// renderStatusIcons builds the leading indicator block for the status bar.
// The connection dot reflects ui.LastError; an empty/nil value renders green.
// Mode tags appear when Vim or plan mode is enabled. The trailing space
// reserves room for a future background-task counter.
func renderStatusIcons() string {
	dot := statusIconOKStyle
	if v := LastError.Load(); v != nil {
		if err, ok := v.(error); ok && err != nil {
			dot = statusIconErrStyle
		}
	}
	out := dot.Render(" ●") + statusBarStyle.Render(" ")

	if VimEnabled.Load() {
		out += statusModeVimStyle.Render("[V]") + statusBarStyle.Render(" ")
	}
	if planActive() {
		out += statusModePlanStyle.Render("[PLAN]") + statusBarStyle.Render(" ")
	}
	// Reserved cell for background task count (not yet wired).
	out += statusBarStyle.Render(" ")
	return out
}

func renderInputLine(prompt, value string, width int) string {
	line := inputPromptStyle.Render(prompt) + value
	if width > 0 && lipgloss.Width(line) < width {
		line += strings.Repeat(" ", width-lipgloss.Width(line))
	}
	return line
}

func renderSeparator(width int) string {
	if width <= 0 {
		return ""
	}
	return separatorStyle.Render(strings.Repeat("─", width))
}

func renderLogs(logs []string, width int) string {
	var b strings.Builder
	for i, ln := range logs {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(thinkingStyle.Render(softWrap(ln, width)))
	}
	return b.String()
}

// renderHelpText builds a one-line-per-command help body for /help. Commands
// are sorted alphabetically and each row is formatted as "/<name> — <synopsis>".
func renderHelpText(reg core.CommandRegistry) string {
	if reg == nil {
		return "(no commands available)"
	}
	cmds := reg.All()
	if len(cmds) == 0 {
		return "(no commands available)"
	}
	// Sort by command name; we don't import sort just for this, rely on a
	// simple insertion-sort over the slice.
	sorted := make([]core.Command, len(cmds))
	copy(sorted, cmds)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j-1].Name() > sorted[j].Name(); j-- {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
		}
	}

	maxName := 0
	for _, c := range sorted {
		if n := len(c.Name()); n > maxName {
			maxName = n
		}
	}

	var b strings.Builder
	for i, c := range sorted {
		if i > 0 {
			b.WriteString("\n")
		}
		name := "/" + c.Name()
		pad := maxName + 1 - len(c.Name())
		if pad < 1 {
			pad = 1
		}
		b.WriteString(name)
		b.WriteString(strings.Repeat(" ", pad))
		b.WriteString("— ")
		b.WriteString(c.Synopsis())
	}
	return b.String()
}
