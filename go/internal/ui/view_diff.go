package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	diffAddStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	diffDelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	diffHunkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Faint(true)

	diffMetaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Bold(true)
)

// renderDiffIfApplicable inspects content for unified-diff markers and, if
// present, returns a colourised rendering with isDiff=true. When no diff
// markers are detected the second return value is false and rendered is the
// empty string — callers should fall back to their default rendering.
func renderDiffIfApplicable(content string, width int) (rendered string, isDiff bool) {
	if !looksLikeDiff(content) {
		return "", false
	}

	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		out = append(out, styleDiffLine(ln, width))
	}
	return strings.Join(out, "\n"), true
}

// looksLikeDiff returns true when content contains enough unified-diff
// signals to be confidently rendered as one. We require at least one of:
// `--- ` plus `+++ ` headers, an `@@` hunk header, or several `+`/`-` lines.
func looksLikeDiff(content string) bool {
	if content == "" {
		return false
	}
	hasMinus := false
	hasPlus := false
	hasHunk := false
	plusLines := 0
	minusLines := 0
	for _, ln := range strings.Split(content, "\n") {
		switch {
		case strings.HasPrefix(ln, "--- "):
			hasMinus = true
		case strings.HasPrefix(ln, "+++ "):
			hasPlus = true
		case strings.HasPrefix(ln, "@@"):
			hasHunk = true
		case strings.HasPrefix(ln, "+") && !strings.HasPrefix(ln, "++"):
			plusLines++
		case strings.HasPrefix(ln, "-") && !strings.HasPrefix(ln, "--"):
			minusLines++
		}
	}
	if hasMinus && hasPlus {
		return true
	}
	if hasHunk {
		return true
	}
	// Heuristic: many +/- prefixed lines (more than three of each) without
	// any obvious negative signals.
	if plusLines >= 3 && minusLines >= 3 {
		return true
	}
	return false
}

// styleDiffLine applies the appropriate style to a single diff line.
func styleDiffLine(line string, width int) string {
	switch {
	case strings.HasPrefix(line, "+++ "), strings.HasPrefix(line, "--- "):
		return diffMetaStyle.Render(line)
	case strings.HasPrefix(line, "@@"):
		return diffHunkStyle.Render(line)
	case strings.HasPrefix(line, "+"):
		return diffAddStyle.Render(line)
	case strings.HasPrefix(line, "-"):
		return diffDelStyle.Render(line)
	default:
		return line
	}
}
