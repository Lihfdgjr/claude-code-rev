package agents

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Definition struct {
	Name         string
	Description  string
	SystemPrompt string
	AllowedTools []string
	Model        string
	MaxTurns     int
	Path         string
}

type Registry struct {
	defs map[string]*Definition
}

func NewRegistry(homeDir, projectDir string) *Registry {
	r := &Registry{defs: map[string]*Definition{}}
	r.defs["general-purpose"] = defaultGeneralPurpose()

	for _, dir := range []string{
		filepath.Join(homeDir, ".claude", "agents"),
		filepath.Join(projectDir, ".claude", "agents"),
	} {
		if dir == "" {
			continue
		}
		loadDir(r, dir)
	}
	return r
}

func (r *Registry) Get(name string) (*Definition, bool) {
	d, ok := r.defs[name]
	return d, ok
}

func (r *Registry) All() []*Definition {
	out := make([]*Definition, 0, len(r.defs))
	for _, n := range r.Names() {
		out = append(out, r.defs[n])
	}
	return out
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.defs))
	for n := range r.defs {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func loadDir(r *Registry, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		def, err := parseFile(path)
		if err != nil || def == nil {
			continue
		}
		if def.Name == "" {
			def.Name = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		}
		r.defs[def.Name] = def
	}
}

func parseFile(path string) (*Definition, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	def := &Definition{Path: path}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var (
		sawOpen   bool
		inFront   bool
		bodyLines []string
	)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimRight(line, "\r")

		if !sawOpen {
			if strings.TrimSpace(trimmed) == "---" {
				sawOpen = true
				inFront = true
				continue
			}
			// no frontmatter — treat entire file as system prompt
			bodyLines = append(bodyLines, trimmed)
			inFront = false
			sawOpen = true
			continue
		}

		if inFront {
			if strings.TrimSpace(trimmed) == "---" {
				inFront = false
				continue
			}
			applyFrontmatter(def, trimmed)
			continue
		}

		bodyLines = append(bodyLines, trimmed)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	def.SystemPrompt = strings.TrimSpace(strings.Join(bodyLines, "\n"))
	return def, nil
}

func applyFrontmatter(def *Definition, line string) {
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return
	}
	key := strings.TrimSpace(strings.ToLower(line[:idx]))
	val := strings.TrimSpace(line[idx+1:])
	val = strings.Trim(val, `"'`)
	if val == "" {
		return
	}
	switch key {
	case "name":
		def.Name = val
	case "description":
		def.Description = val
	case "model":
		def.Model = val
	case "tools":
		def.AllowedTools = splitTools(val)
	case "max_turns", "maxturns":
		def.MaxTurns = parseInt(val)
	}
}

func splitTools(v string) []string {
	fields := strings.FieldsFunc(v, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func defaultGeneralPurpose() *Definition {
	return &Definition{
		Name:        "general-purpose",
		Description: "General-purpose sub-agent for research, analysis, and self-contained tasks.",
		SystemPrompt: "You are a general-purpose sub-agent. You have been spawned to investigate or carry out a focused task on behalf of the parent agent. Use the tools available to you to gather information, perform the work, and verify your results. Be thorough but stay on task. When you finish, return a concise final response that the parent agent can act on directly; do not include filler, plans, or status updates beyond what the parent needs.",
	}
}
