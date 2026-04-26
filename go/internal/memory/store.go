package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Entry is one persisted memory file under ~/.claude/memory/.
type Entry struct {
	Name        string // basename without .md
	Type        string // user | feedback | project | reference
	Description string
	Body        string
	Path        string
	Modified    time.Time
}

// Store manages the per-type memory files plus a MEMORY.md index.
type Store struct {
	root string
}

func NewStore(homeDir string) *Store {
	root := filepath.Join(homeDir, ".claude", "memory")
	_ = os.MkdirAll(root, 0o755)
	return &Store{root: root}
}

func (s *Store) Root() string { return s.root }

func (s *Store) List() ([]*Entry, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]*Entry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if e.Name() == "MEMORY.md" {
			continue
		}
		full := filepath.Join(s.root, e.Name())
		ent, err := s.readEntry(full)
		if err != nil {
			continue
		}
		out = append(out, ent)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Modified.After(out[j].Modified)
	})
	return out, nil
}

func (s *Store) Get(name string) (*Entry, error) {
	path := filepath.Join(s.root, sanitize(name)+".md")
	return s.readEntry(path)
}

func (s *Store) Save(e *Entry) error {
	if e == nil || e.Name == "" {
		return fmt.Errorf("memory: empty entry name")
	}
	if e.Type == "" {
		e.Type = "project"
	}
	path := filepath.Join(s.root, sanitize(e.Name)+".md")
	body := strings.TrimSpace(e.Body)
	doc := fmt.Sprintf("---\nname: %s\ndescription: %s\ntype: %s\n---\n\n%s\n",
		e.Name, oneLine(e.Description), e.Type, body)
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(doc), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	e.Path = path
	e.Modified = time.Now()
	return s.writeIndex()
}

func (s *Store) Delete(name string) error {
	path := filepath.Join(s.root, sanitize(name)+".md")
	if err := os.Remove(path); err != nil {
		return err
	}
	return s.writeIndex()
}

func (s *Store) writeIndex() error {
	entries, err := s.List()
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# Memory index\n\n")
	if len(entries) == 0 {
		b.WriteString("(empty)\n")
	}
	for _, e := range entries {
		fmt.Fprintf(&b, "- [%s](%s.md) — %s — %s\n", e.Name, sanitize(e.Name), e.Type, oneLine(e.Description))
	}
	return os.WriteFile(filepath.Join(s.root, "MEMORY.md"), []byte(b.String()), 0o644)
}

var frontmatterRE = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n?(.*)$`)

func (s *Store) readEntry(path string) (*Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	st, _ := os.Stat(path)
	e := &Entry{
		Path:     path,
		Name:     strings.TrimSuffix(filepath.Base(path), ".md"),
		Modified: st.ModTime(),
	}
	if m := frontmatterRE.FindStringSubmatch(string(data)); m != nil {
		for _, line := range strings.Split(m[1], "\n") {
			k, v, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			switch k {
			case "name":
				if v != "" {
					e.Name = v
				}
			case "description":
				e.Description = v
			case "type":
				e.Type = v
			}
		}
		e.Body = strings.TrimSpace(m[2])
	} else {
		e.Body = strings.TrimSpace(string(data))
	}
	if e.Type == "" {
		e.Type = "project"
	}
	return e, nil
}

var nameSafe = regexp.MustCompile(`[^A-Za-z0-9_\-]+`)

func sanitize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nameSafe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "entry"
	}
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}
