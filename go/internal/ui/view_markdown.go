package ui

import (
	"regexp"
	"strings"
)

var (
	mdH1Re   = regexp.MustCompile(`^# (.+)$`)
	mdH2Re   = regexp.MustCompile(`^## (.+)$`)
	mdH3Re   = regexp.MustCompile(`^### (.+)$`)
	mdULRe   = regexp.MustCompile(`^(\s*)([-*]) (.+)$`)
	mdOLRe   = regexp.MustCompile(`^(\s*)(\d+\.) (.+)$`)
	mdQuote  = regexp.MustCompile(`^> (.*)$`)
	mdBoldA  = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	mdBoldB  = regexp.MustCompile(`__([^_]+)__`)
	mdItalA  = regexp.MustCompile(`(^|[^A-Za-z0-9_])\*([^*\s][^*]*?)\*([^A-Za-z0-9_]|$)`)
	mdItalB  = regexp.MustCompile(`(^|[^A-Za-z0-9_])_([^_\s][^_]*?)_([^A-Za-z0-9_]|$)`)
	mdCode   = regexp.MustCompile("`([^`]+)`")
	mdLink   = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

// renderMarkdown applies basic inline + block markdown styling. Fenced code
// blocks (``` ... ```) are passed through verbatim so the existing fenced
// highlighter can colour them downstream.
func renderMarkdown(text string) string {
	if text == "" {
		return text
	}
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	inFence := false
	for _, ln := range lines {
		trim := strings.TrimSpace(ln)
		if strings.HasPrefix(trim, "```") {
			inFence = !inFence
			out = append(out, ln)
			continue
		}
		if inFence {
			out = append(out, ln)
			continue
		}
		out = append(out, renderMarkdownLine(ln))
	}
	return strings.Join(out, "\n")
}

func renderMarkdownLine(ln string) string {
	if m := mdH1Re.FindStringSubmatch(ln); m != nil {
		return markdownH1Style.Render(m[1])
	}
	if m := mdH2Re.FindStringSubmatch(ln); m != nil {
		return markdownH2Style.Render(m[1])
	}
	if m := mdH3Re.FindStringSubmatch(ln); m != nil {
		return markdownH3Style.Render(m[1])
	}
	if m := mdQuote.FindStringSubmatch(ln); m != nil {
		body := applyMarkdownInline(m[1])
		return markdownQuoteStyle.Render("│ ") + markdownQuoteStyle.Render(body)
	}
	if m := mdULRe.FindStringSubmatch(ln); m != nil {
		return m[1] + markdownListPrefixStyle.Render(m[2]+" ") + applyMarkdownInline(m[3])
	}
	if m := mdOLRe.FindStringSubmatch(ln); m != nil {
		return m[1] + markdownListPrefixStyle.Render(m[2]+" ") + applyMarkdownInline(m[3])
	}
	return applyMarkdownInline(ln)
}

func applyMarkdownInline(s string) string {
	s = mdCode.ReplaceAllStringFunc(s, func(m string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(m, "`"), "`")
		return markdownCodeStyle.Render(inner)
	})
	s = mdBoldA.ReplaceAllStringFunc(s, func(m string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(m, "**"), "**")
		return markdownBoldStyle.Render(inner)
	})
	s = mdBoldB.ReplaceAllStringFunc(s, func(m string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(m, "__"), "__")
		return markdownBoldStyle.Render(inner)
	})
	s = mdItalA.ReplaceAllStringFunc(s, func(m string) string {
		sub := mdItalA.FindStringSubmatch(m)
		if sub == nil {
			return m
		}
		return sub[1] + markdownItalicStyle.Render(sub[2]) + sub[3]
	})
	s = mdItalB.ReplaceAllStringFunc(s, func(m string) string {
		sub := mdItalB.FindStringSubmatch(m)
		if sub == nil {
			return m
		}
		return sub[1] + markdownItalicStyle.Render(sub[2]) + sub[3]
	})
	s = mdLink.ReplaceAllStringFunc(s, func(m string) string {
		sub := mdLink.FindStringSubmatch(m)
		if sub == nil {
			return m
		}
		return markdownLinkStyle.Render(sub[1]) + " " + markdownQuoteStyle.Render("("+sub[2]+")")
	})
	return s
}
