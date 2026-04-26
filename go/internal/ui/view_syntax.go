package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	keywordStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14"))

	stringStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	commentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true)
)

// language keyword tables. Kept small and pragmatic — enough to give a
// recogniseable look without trying to be a real lexer.
var langKeywords = map[string]map[string]bool{
	"go": {
		"break": true, "case": true, "chan": true, "const": true, "continue": true,
		"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
		"func": true, "go": true, "goto": true, "if": true, "import": true,
		"interface": true, "map": true, "package": true, "range": true, "return": true,
		"select": true, "struct": true, "switch": true, "type": true, "var": true,
		"true": true, "false": true, "nil": true,
	},
	"python": {
		"def": true, "class": true, "return": true, "if": true, "elif": true,
		"else": true, "for": true, "while": true, "try": true, "except": true,
		"finally": true, "raise": true, "import": true, "from": true, "as": true,
		"with": true, "yield": true, "lambda": true, "pass": true, "break": true,
		"continue": true, "global": true, "nonlocal": true, "in": true, "is": true,
		"not": true, "and": true, "or": true, "True": true, "False": true,
		"None": true, "async": true, "await": true,
	},
	"javascript": {
		"var": true, "let": true, "const": true, "function": true, "return": true,
		"if": true, "else": true, "for": true, "while": true, "do": true,
		"switch": true, "case": true, "default": true, "break": true, "continue": true,
		"new": true, "delete": true, "typeof": true, "instanceof": true, "in": true,
		"of": true, "class": true, "extends": true, "super": true, "this": true,
		"import": true, "export": true, "from": true, "as": true, "async": true,
		"await": true, "try": true, "catch": true, "finally": true, "throw": true,
		"true": true, "false": true, "null": true, "undefined": true,
		"interface": true, "type": true, "enum": true, "implements": true,
	},
	"bash": {
		"if": true, "then": true, "else": true, "elif": true, "fi": true,
		"for": true, "while": true, "do": true, "done": true, "case": true,
		"esac": true, "in": true, "function": true, "return": true, "echo": true,
		"export": true, "local": true, "readonly": true, "set": true, "unset": true,
		"shift": true, "test": true, "true": true, "false": true, "exit": true,
	},
}

// commentToken describes which prefix introduces a single-line comment for
// each language family.
var langComment = map[string]string{
	"go":         "//",
	"python":     "#",
	"javascript": "//",
	"bash":       "#",
}

// normaliseLang collapses aliases to a canonical key in langKeywords.
func normaliseLang(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "go", "golang":
		return "go"
	case "py", "python":
		return "python"
	case "js", "ts", "javascript", "typescript", "jsx", "tsx":
		return "javascript"
	case "sh", "bash", "shell", "zsh":
		return "bash"
	}
	return ""
}

// highlightCode applies keyword/string/comment highlighting to code in the
// given language. Unknown languages are returned untouched. Highlighting is
// line-by-line and intentionally simple — multi-line strings or nested
// constructs are not handled.
func highlightCode(code, lang string) string {
	canon := normaliseLang(lang)
	if canon == "" {
		return code
	}

	keywords := langKeywords[canon]
	commentTok := langComment[canon]

	lines := strings.Split(code, "\n")
	out := make([]string, len(lines))
	for i, ln := range lines {
		out[i] = highlightLine(ln, keywords, commentTok)
	}
	return strings.Join(out, "\n")
}

// highlightLine processes a single source line. It walks the line once,
// extracting strings (between matching " or ' quotes) and comments first
// then highlighting bare identifier runs that match the keyword table.
func highlightLine(line string, keywords map[string]bool, commentTok string) string {
	var b strings.Builder
	runes := []rune(line)
	i := 0
	for i < len(runes) {
		// Comment: from here to end of line.
		if commentTok != "" && hasPrefixAt(runes, i, commentTok) {
			b.WriteString(commentStyle.Render(string(runes[i:])))
			return b.String()
		}

		c := runes[i]
		// String literal: scan to matching closing quote, handling \" escapes.
		if c == '"' || c == '\'' || c == '`' {
			end := i + 1
			for end < len(runes) {
				if runes[end] == '\\' && end+1 < len(runes) {
					end += 2
					continue
				}
				if runes[end] == c {
					end++
					break
				}
				end++
			}
			b.WriteString(stringStyle.Render(string(runes[i:end])))
			i = end
			continue
		}

		// Identifier run: letters/digits/underscore.
		if isIdentStart(c) {
			end := i + 1
			for end < len(runes) && isIdentCont(runes[end]) {
				end++
			}
			word := string(runes[i:end])
			if keywords[word] {
				b.WriteString(keywordStyle.Render(word))
			} else {
				b.WriteString(word)
			}
			i = end
			continue
		}

		b.WriteRune(c)
		i++
	}
	return b.String()
}

func hasPrefixAt(runes []rune, idx int, prefix string) bool {
	pr := []rune(prefix)
	if idx+len(pr) > len(runes) {
		return false
	}
	for j, r := range pr {
		if runes[idx+j] != r {
			return false
		}
	}
	return true
}

func isIdentStart(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isIdentCont(r rune) bool {
	return isIdentStart(r) || (r >= '0' && r <= '9')
}

// highlightFencedBlocks scans text for triple-backtick fenced code blocks and
// rewrites the body with highlightCode for the declared language. Blocks
// without a language tag are left as-is; the surrounding fence markers are
// preserved verbatim.
func highlightFencedBlocks(text string) string {
	if !strings.Contains(text, "```") {
		return text
	}

	lines := strings.Split(text, "\n")
	var b strings.Builder

	inBlock := false
	var lang string
	var block strings.Builder

	flush := func(trailingNewline bool) {
		body := block.String()
		// Drop trailing newline added by the join below.
		body = strings.TrimRight(body, "\n")
		if lang != "" {
			body = highlightCode(body, lang)
		}
		b.WriteString(body)
		if trailingNewline {
			b.WriteString("\n")
		}
		block.Reset()
		lang = ""
	}

	for i, ln := range lines {
		isFence := strings.HasPrefix(strings.TrimSpace(ln), "```")
		if !inBlock {
			if isFence {
				inBlock = true
				lang = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ln), "```"))
				b.WriteString(ln)
				if i < len(lines)-1 {
					b.WriteString("\n")
				}
				continue
			}
			b.WriteString(ln)
			if i < len(lines)-1 {
				b.WriteString("\n")
			}
			continue
		}
		// Inside a fenced block.
		if isFence {
			flush(false)
			b.WriteString("\n")
			b.WriteString(ln)
			if i < len(lines)-1 {
				b.WriteString("\n")
			}
			inBlock = false
			continue
		}
		block.WriteString(ln)
		block.WriteString("\n")
	}
	if inBlock {
		// Unterminated block: emit the buffered body unhighlighted-safe.
		flush(false)
	}
	return b.String()
}
