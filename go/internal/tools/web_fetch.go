package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"claudecode/internal/core"
)

type webFetchTool struct{}

func NewWebFetch() core.Tool { return &webFetchTool{} }

func (t *webFetchTool) Name() string { return "WebFetch" }

func (t *webFetchTool) Description() string {
	return "HTTP GET a URL with a 30s timeout. Follows up to 5 redirects, reads up to 1 MiB. Converts HTML responses to Markdown. Caps output at 50000 chars."
}

func (t *webFetchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"url": {"type": "string"},
			"prompt": {"type": "string"}
		},
		"required": ["url"],
		"additionalProperties": false
	}`)
}

var (
	scriptRE   = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleRE    = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	noscriptRE = regexp.MustCompile(`(?is)<noscript[^>]*>.*?</noscript>`)
	wsRE       = regexp.MustCompile(`[ \t]+`)
	nlRE       = regexp.MustCompile(`\n{3,}`)
)

// htmlToMarkdown converts a tiny subset of HTML to Markdown using a single
// linear scan. Unknown tags are dropped but their text content is preserved.
func htmlToMarkdown(s string) string {
	s = scriptRE.ReplaceAllString(s, "")
	s = styleRE.ReplaceAllString(s, "")
	s = noscriptRE.ReplaceAllString(s, "")

	var out strings.Builder
	out.Grow(len(s))

	// Stack of currently-open formatting wrappers we need to mirror on close.
	type frame struct {
		tag      string
		listKind string // "ul", "ol", "" otherwise
		olIndex  int    // running counter for ordered lists
	}
	var listStack []frame

	currentList := func() *frame {
		for i := len(listStack) - 1; i >= 0; i-- {
			if listStack[i].listKind != "" {
				return &listStack[i]
			}
		}
		return nil
	}

	i := 0
	for i < len(s) {
		c := s[i]
		if c != '<' {
			// Accumulate text up to next tag.
			j := strings.IndexByte(s[i:], '<')
			var chunk string
			if j < 0 {
				chunk = s[i:]
				i = len(s)
			} else {
				chunk = s[i : i+j]
				i += j
			}
			out.WriteString(decodeEntities(chunk))
			continue
		}
		// Find tag end.
		end := strings.IndexByte(s[i:], '>')
		if end < 0 {
			out.WriteByte(c)
			i++
			continue
		}
		raw := s[i+1 : i+end]
		i += end + 1

		// Comments.
		if strings.HasPrefix(raw, "!--") {
			continue
		}

		closing := false
		if strings.HasPrefix(raw, "/") {
			closing = true
			raw = raw[1:]
		}
		raw = strings.TrimSpace(raw)
		raw = strings.TrimSuffix(raw, "/")
		raw = strings.TrimSpace(raw)

		name, attrs := splitTag(raw)
		name = strings.ToLower(name)

		switch name {
		case "br":
			out.WriteByte('\n')
		case "hr":
			out.WriteString("\n\n---\n\n")
		case "p", "div":
			if !closing {
				ensureBlankLine(&out)
			} else {
				ensureBlankLine(&out)
			}
		case "h1", "h2", "h3", "h4", "h5", "h6":
			level, _ := strconv.Atoi(name[1:])
			if !closing {
				ensureBlankLine(&out)
				out.WriteString(strings.Repeat("#", level))
				out.WriteByte(' ')
			} else {
				out.WriteByte('\n')
				ensureBlankLine(&out)
			}
		case "strong", "b":
			out.WriteString("**")
		case "em", "i":
			out.WriteByte('*')
		case "code":
			// Inline code; pre handles fenced blocks.
			out.WriteByte('`')
		case "pre":
			if !closing {
				ensureBlankLine(&out)
				out.WriteString("```\n")
			} else {
				out.WriteString("\n```\n")
			}
		case "blockquote":
			if !closing {
				ensureBlankLine(&out)
				out.WriteString("> ")
			} else {
				ensureBlankLine(&out)
			}
		case "ul":
			if !closing {
				listStack = append(listStack, frame{tag: name, listKind: "ul"})
				ensureBlankLine(&out)
			} else if len(listStack) > 0 {
				listStack = listStack[:len(listStack)-1]
				ensureBlankLine(&out)
			}
		case "ol":
			if !closing {
				listStack = append(listStack, frame{tag: name, listKind: "ol"})
				ensureBlankLine(&out)
			} else if len(listStack) > 0 {
				listStack = listStack[:len(listStack)-1]
				ensureBlankLine(&out)
			}
		case "li":
			if !closing {
				out.WriteByte('\n')
				if cur := currentList(); cur != nil && cur.listKind == "ol" {
					cur.olIndex++
					out.WriteString(strconv.Itoa(cur.olIndex))
					out.WriteString(". ")
				} else {
					out.WriteString("- ")
				}
			}
		case "a":
			if !closing {
				href := attrValue(attrs, "href")
				if href != "" {
					out.WriteByte('[')
					// Stash href on a tiny stack via a sentinel: write closing later
					// using a marker; simplest is to encode now and push state.
					listStack = append(listStack, frame{tag: "a:" + href})
				}
			} else {
				// Find matching open frame.
				for k := len(listStack) - 1; k >= 0; k-- {
					if strings.HasPrefix(listStack[k].tag, "a:") {
						href := strings.TrimPrefix(listStack[k].tag, "a:")
						out.WriteString("](")
						out.WriteString(href)
						out.WriteByte(')')
						listStack = append(listStack[:k], listStack[k+1:]...)
						break
					}
				}
			}
		case "img":
			alt := attrValue(attrs, "alt")
			src := attrValue(attrs, "src")
			if src != "" {
				out.WriteString("![")
				out.WriteString(alt)
				out.WriteString("](")
				out.WriteString(src)
				out.WriteByte(')')
			}
		default:
			// Unknown tag: drop the tag itself, keep text content intact.
		}
	}

	text := out.String()
	text = wsRE.ReplaceAllString(text, " ")
	text = nlRE.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

// ensureBlankLine writes up to two newlines so the next block starts on a
// fresh paragraph without piling on extras.
func ensureBlankLine(b *strings.Builder) {
	s := b.String()
	switch {
	case len(s) == 0:
		return
	case strings.HasSuffix(s, "\n\n"):
		return
	case strings.HasSuffix(s, "\n"):
		b.WriteByte('\n')
	default:
		b.WriteString("\n\n")
	}
}

// splitTag splits "<tag attr=...>" inner contents into name and attrs string.
func splitTag(raw string) (name, attrs string) {
	for i := 0; i < len(raw); i++ {
		if unicode.IsSpace(rune(raw[i])) {
			return raw[:i], strings.TrimSpace(raw[i+1:])
		}
	}
	return raw, ""
}

// attrValue extracts a single attribute value from an attr blob. Handles
// quoted ("..." / '...') and bare values.
func attrValue(attrs, key string) string {
	lower := strings.ToLower(attrs)
	k := strings.ToLower(key)
	idx := 0
	for {
		pos := strings.Index(lower[idx:], k)
		if pos < 0 {
			return ""
		}
		start := idx + pos
		// Must be at start or preceded by whitespace.
		if start > 0 && !unicode.IsSpace(rune(attrs[start-1])) {
			idx = start + len(k)
			continue
		}
		end := start + len(k)
		// Skip whitespace then expect '='.
		for end < len(attrs) && unicode.IsSpace(rune(attrs[end])) {
			end++
		}
		if end >= len(attrs) || attrs[end] != '=' {
			idx = start + len(k)
			continue
		}
		end++
		for end < len(attrs) && unicode.IsSpace(rune(attrs[end])) {
			end++
		}
		if end >= len(attrs) {
			return ""
		}
		quote := attrs[end]
		if quote == '"' || quote == '\'' {
			end++
			closeIdx := strings.IndexByte(attrs[end:], quote)
			if closeIdx < 0 {
				return decodeEntities(attrs[end:])
			}
			return decodeEntities(attrs[end : end+closeIdx])
		}
		// Bare value up to whitespace.
		stop := end
		for stop < len(attrs) && !unicode.IsSpace(rune(attrs[stop])) {
			stop++
		}
		return decodeEntities(attrs[end:stop])
	}
}

// decodeEntities resolves the minimal HTML entity set we care about.
func decodeEntities(s string) string {
	if !strings.ContainsRune(s, '&') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] != '&' {
			b.WriteByte(s[i])
			i++
			continue
		}
		semi := strings.IndexByte(s[i:], ';')
		if semi < 0 || semi > 10 {
			b.WriteByte('&')
			i++
			continue
		}
		entity := s[i+1 : i+semi]
		switch entity {
		case "amp":
			b.WriteByte('&')
		case "lt":
			b.WriteByte('<')
		case "gt":
			b.WriteByte('>')
		case "quot":
			b.WriteByte('"')
		case "apos":
			b.WriteByte('\'')
		case "nbsp":
			b.WriteByte(' ')
		default:
			if strings.HasPrefix(entity, "#") {
				num := entity[1:]
				base := 10
				if strings.HasPrefix(num, "x") || strings.HasPrefix(num, "X") {
					num = num[1:]
					base = 16
				}
				if v, err := strconv.ParseInt(num, base, 32); err == nil && v > 0 {
					b.WriteRune(rune(v))
				} else {
					b.WriteString(s[i : i+semi+1])
				}
			} else {
				b.WriteString(s[i : i+semi+1])
			}
		}
		i += semi + 1
	}
	return b.String()
}

func (t *webFetchTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		URL    string `json:"url"`
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if args.URL == "" {
		return "", fmt.Errorf("url required")
	}

	redirects := 0
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirects++
			if redirects > 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", args.URL, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "claudecode-webfetch/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, 1<<20)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	text := string(body)
	if strings.Contains(strings.ToLower(ct), "html") {
		text = htmlToMarkdown(text)
	}
	if len(text) > 50000 {
		text = text[:50000] + "\n...[truncated to 50000 chars]"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "URL: %s\nContent-Type: %s\n\n%s", args.URL, ct, text)
	if args.Prompt != "" {
		fmt.Fprintf(&b, "\n\n[user prompt hint: %s]", args.Prompt)
	}
	return b.String(), nil
}
