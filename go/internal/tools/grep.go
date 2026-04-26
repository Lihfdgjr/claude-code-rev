package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"claudecode/internal/core"
)

type grepTool struct{}

type grepInput struct {
	Pattern         string `json:"pattern"`
	Path            string `json:"path,omitempty"`
	Glob            string `json:"glob,omitempty"`
	Type            string `json:"type,omitempty"`
	CaseInsensitive bool   `json:"case_insensitive,omitempty"`
	OutputMode      string `json:"output_mode,omitempty"`
	HeadLimit       int    `json:"head_limit,omitempty"`
}

func NewGrep() core.Tool { return &grepTool{} }

func (grepTool) Name() string { return "Grep" }

func (grepTool) Description() string {
	return "Search files for a regular expression pattern. Supports glob/type filters and three output modes: files_with_matches, content, count."
}

func (grepTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "pattern": {"type": "string", "description": "Regular expression to search for"},
    "path": {"type": "string", "description": "File or directory to search (default '.')"},
    "glob": {"type": "string", "description": "Glob pattern to filter file names (e.g. '*.go')"},
    "type": {"type": "string", "description": "File type alias (js, ts, go, py, rust, java)"},
    "case_insensitive": {"type": "boolean", "description": "Case-insensitive match"},
    "output_mode": {"type": "string", "enum": ["files_with_matches", "content", "count"], "description": "Output format"},
    "head_limit": {"type": "integer", "description": "Limit number of result entries", "minimum": 1}
  },
  "required": ["pattern"],
  "additionalProperties": false
}`)
}

var grepTypeGlobs = map[string][]string{
	"js":   {"*.js", "*.jsx"},
	"ts":   {"*.ts", "*.tsx"},
	"go":   {"*.go"},
	"py":   {"*.py"},
	"rust": {"*.rs"},
	"java": {"*.java"},
}

func (grepTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in grepInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if in.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	patternStr := in.Pattern
	if in.CaseInsensitive {
		patternStr = "(?i)" + patternStr
	}
	re, err := regexp.Compile(patternStr)
	if err != nil {
		return "", fmt.Errorf("invalid pattern: %w", err)
	}

	root := in.Path
	if root == "" {
		root = "."
	}
	mode := in.OutputMode
	if mode == "" {
		mode = "files_with_matches"
	}
	switch mode {
	case "files_with_matches", "content", "count":
	default:
		return "", fmt.Errorf("invalid output_mode: %s", mode)
	}

	var typeGlobs []string
	if in.Type != "" {
		g, ok := grepTypeGlobs[strings.ToLower(in.Type)]
		if !ok {
			return "", fmt.Errorf("unknown type: %s", in.Type)
		}
		typeGlobs = g
	}

	matchesName := func(name string) bool {
		if in.Glob != "" {
			ok, err := filepath.Match(in.Glob, name)
			if err == nil && ok {
				return true
			}
			return false
		}
		if len(typeGlobs) > 0 {
			for _, g := range typeGlobs {
				if ok, _ := filepath.Match(g, name); ok {
					return true
				}
			}
			return false
		}
		return true
	}

	type fileResult struct {
		path   string
		count  int
		lines  []string
		linums []int
	}
	var results []fileResult

	walk := func(p string, info os.FileInfo) error {
		if info.IsDir() {
			return nil
		}
		if !matchesName(filepath.Base(p)) {
			return nil
		}
		f, err := os.Open(p)
		if err != nil {
			return nil
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
		var fr fileResult
		fr.path = p
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			if re.MatchString(line) {
				fr.count++
				if mode == "content" {
					fr.lines = append(fr.lines, line)
					fr.linums = append(fr.linums, lineNo)
				}
			}
		}
		if fr.count > 0 {
			results = append(results, fr)
		}
		return nil
	}

	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		err = filepath.Walk(root, func(p string, fi os.FileInfo, werr error) error {
			if werr != nil {
				return nil
			}
			return walk(p, fi)
		})
		if err != nil {
			return "", err
		}
	} else {
		if err := walk(root, info); err != nil {
			return "", err
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].path < results[j].path })

	limit := in.HeadLimit
	var b strings.Builder
	emitted := 0
	switch mode {
	case "files_with_matches":
		for _, r := range results {
			if limit > 0 && emitted >= limit {
				break
			}
			b.WriteString(r.path)
			b.WriteByte('\n')
			emitted++
		}
	case "count":
		for _, r := range results {
			if limit > 0 && emitted >= limit {
				break
			}
			fmt.Fprintf(&b, "%s:%d\n", r.path, r.count)
			emitted++
		}
	case "content":
		for _, r := range results {
			for i, line := range r.lines {
				if limit > 0 && emitted >= limit {
					break
				}
				fmt.Fprintf(&b, "%s:%d:%s\n", r.path, r.linums[i], line)
				emitted++
			}
			if limit > 0 && emitted >= limit {
				break
			}
		}
	}
	return b.String(), nil
}
