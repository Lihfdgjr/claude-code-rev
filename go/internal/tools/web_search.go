package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"claudecode/internal/core"
)

type webSearchTool struct{}

func NewWebSearch() core.Tool { return &webSearchTool{} }

func (t *webSearchTool) Name() string { return "WebSearch" }

func (t *webSearchTool) Description() string {
	return "Search the web via DuckDuckGo HTML endpoint. Returns top 10 results with title, URL, and snippet. Supports allowed_domains/blocked_domains filters. 15s timeout."
}

func (t *webSearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {"type": "string"},
			"allowed_domains": {
				"type": "array",
				"items": {"type": "string"}
			},
			"blocked_domains": {
				"type": "array",
				"items": {"type": "string"}
			}
		},
		"required": ["query"],
		"additionalProperties": false
	}`)
}

var (
	resultLinkRE = regexp.MustCompile(`(?is)<a\s+[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	snippetRE    = regexp.MustCompile(`(?is)<a\s+[^>]*class="[^"]*result__snippet[^"]*"[^>]*>(.*?)</a>`)
	resultBlock  = regexp.MustCompile(`(?is)<div\s+[^>]*class="[^"]*result\b[^"]*"[^>]*>(.*?)</div>\s*</div>`)
	htmlTagRE    = regexp.MustCompile(`(?s)<[^>]+>`)
)

func cleanHTMLText(s string) string {
	s = htmlTagRE.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&#x27;", "'")
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

// resolveDDGURL unwraps DuckDuckGo's redirect wrapper (//duckduckgo.com/l/?uddg=...).
func resolveDDGURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if strings.Contains(u.Host, "duckduckgo.com") && strings.HasPrefix(u.Path, "/l/") {
		if v := u.Query().Get("uddg"); v != "" {
			return v
		}
	}
	return raw
}

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func domainMatch(host, pattern string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if host == "" || pattern == "" {
		return false
	}
	if host == pattern {
		return true
	}
	return strings.HasSuffix(host, "."+pattern)
}

type searchResult struct {
	Title   string
	URL     string
	Snippet string
}

func parseDDGResults(html string) []searchResult {
	var out []searchResult
	blocks := resultBlock.FindAllStringSubmatch(html, -1)
	if len(blocks) == 0 {
		// Fallback: scan link/snippet pairs across the whole page.
		links := resultLinkRE.FindAllStringSubmatch(html, -1)
		snips := snippetRE.FindAllStringSubmatch(html, -1)
		for i, l := range links {
			r := searchResult{
				URL:   resolveDDGURL(l[1]),
				Title: cleanHTMLText(l[2]),
			}
			if i < len(snips) {
				r.Snippet = cleanHTMLText(snips[i][1])
			}
			out = append(out, r)
		}
		return out
	}
	for _, b := range blocks {
		body := b[1]
		linkMatch := resultLinkRE.FindStringSubmatch(body)
		if linkMatch == nil {
			continue
		}
		r := searchResult{
			URL:   resolveDDGURL(linkMatch[1]),
			Title: cleanHTMLText(linkMatch[2]),
		}
		if sm := snippetRE.FindStringSubmatch(body); sm != nil {
			r.Snippet = cleanHTMLText(sm[1])
		}
		out = append(out, r)
	}
	return out
}

func (t *webSearchTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Query          string   `json:"query"`
		AllowedDomains []string `json:"allowed_domains"`
		BlockedDomains []string `json:"blocked_domains"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if strings.TrimSpace(args.Query) == "" {
		return "", fmt.Errorf("query required")
	}

	endpoint := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(args.Query)
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Gecko/20100101 Firefox/115.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("search request failed with status %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	results := parseDDGResults(string(body))
	if len(results) == 0 {
		return fmt.Sprintf("No results found for %q.", args.Query), nil
	}

	var filtered []searchResult
	for _, r := range results {
		host := hostOf(r.URL)
		if host == "" {
			continue
		}
		if len(args.AllowedDomains) > 0 {
			ok := false
			for _, d := range args.AllowedDomains {
				if domainMatch(host, d) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		if len(args.BlockedDomains) > 0 {
			blocked := false
			for _, d := range args.BlockedDomains {
				if domainMatch(host, d) {
					blocked = true
					break
				}
			}
			if blocked {
				continue
			}
		}
		filtered = append(filtered, r)
		if len(filtered) >= 10 {
			break
		}
	}

	if len(filtered) == 0 {
		return fmt.Sprintf("No results for %q after applying domain filters (%d unfiltered hits).", args.Query, len(results)), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Web search results for %q:\n\n", args.Query)
	for i, r := range filtered {
		snippet := r.Snippet
		if snippet == "" {
			snippet = "(no snippet)"
		}
		fmt.Fprintf(&b, "%d. %s\n   %s\n   %s\n\n", i+1, r.Title, r.URL, snippet)
	}
	return strings.TrimRight(b.String(), "\n") + "\n", nil
}
