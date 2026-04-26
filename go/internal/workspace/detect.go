package workspace

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Workspace struct {
	Root           string
	Kind           string
	Languages      []string
	HasGoMod       bool
	HasPackageJSON bool
	HasCargoToml   bool
	HasPyProject   bool
}

var manifestNames = []string{
	"go.mod",
	"package.json",
	"Cargo.toml",
	"pyproject.toml",
	"requirements.txt",
}

var extLang = map[string]string{
	".go":    "Go",
	".py":    "Python",
	".ts":    "TypeScript",
	".tsx":   "TypeScript",
	".js":    "JavaScript",
	".jsx":   "JavaScript",
	".rs":    "Rust",
	".java":  "Java",
	".kt":    "Kotlin",
	".rb":    "Ruby",
	".cpp":   "C++",
	".cc":    "C++",
	".c":     "C",
	".h":     "C",
	".hpp":   "C++",
	".cs":    "C#",
	".swift": "Swift",
	".php":   "PHP",
	".sh":    "Shell",
	".lua":   "Lua",
	".dart":  "Dart",
	".scala": "Scala",
	".ex":    "Elixir",
	".exs":   "Elixir",
}

func Detect(startDir string) *Workspace {
	w := &Workspace{Kind: "none"}
	if startDir == "" {
		startDir, _ = os.Getwd()
	}
	abs, err := filepath.Abs(startDir)
	if err != nil {
		abs = startDir
	}
	w.Root = abs

	dir := abs
	for {
		if st, err := os.Stat(filepath.Join(dir, ".git")); err == nil && st.IsDir() {
			w.Root = dir
			w.Kind = "git"
			break
		}
		if st, err := os.Stat(filepath.Join(dir, ".hg")); err == nil && st.IsDir() {
			w.Root = dir
			w.Kind = "hg"
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	for _, m := range manifestNames {
		if _, err := os.Stat(filepath.Join(w.Root, m)); err == nil {
			switch m {
			case "go.mod":
				w.HasGoMod = true
			case "package.json":
				w.HasPackageJSON = true
			case "Cargo.toml":
				w.HasCargoToml = true
			case "pyproject.toml":
				w.HasPyProject = true
			}
		}
	}

	w.Languages = topLanguages(w.Root)
	return w
}

func topLanguages(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	if len(entries) > 100 {
		entries = entries[:100]
	}
	counts := map[string]int{}
	for _, e := range entries {
		if e.IsDir() {
			sub, err := os.ReadDir(filepath.Join(root, e.Name()))
			if err != nil {
				continue
			}
			limit := len(sub)
			if limit > 100 {
				limit = 100
			}
			for i := 0; i < limit; i++ {
				if sub[i].IsDir() {
					continue
				}
				ext := strings.ToLower(filepath.Ext(sub[i].Name()))
				if lang, ok := extLang[ext]; ok {
					counts[lang]++
				}
			}
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if lang, ok := extLang[ext]; ok {
			counts[lang]++
		}
	}
	type kv struct {
		Lang  string
		Count int
	}
	pairs := make([]kv, 0, len(counts))
	for l, c := range counts {
		pairs = append(pairs, kv{l, c})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Count != pairs[j].Count {
			return pairs[i].Count > pairs[j].Count
		}
		return pairs[i].Lang < pairs[j].Lang
	})
	if len(pairs) > 3 {
		pairs = pairs[:3]
	}
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, p.Lang)
	}
	return out
}
