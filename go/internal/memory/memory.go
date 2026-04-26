package memory

import (
	"os"
	"path/filepath"
	"strings"
)

type FileMemory struct {
	Path    string
	Content string
}

type Memory struct {
	ProjectFiles []FileMemory
	UserFile     *FileMemory
}

func LoadProject(projectDir string) *Memory {
	m := &Memory{}

	if projectDir != "" {
		dir, err := filepath.Abs(projectDir)
		if err == nil {
			for {
				p := filepath.Join(dir, "CLAUDE.md")
				if fm, ok := readFile(p); ok {
					m.ProjectFiles = append(m.ProjectFiles, fm)
				}
				parent := filepath.Dir(dir)
				if parent == dir {
					break
				}
				dir = parent
			}
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		userPath := filepath.Join(home, ".claude", "CLAUDE.md")
		if fm, ok := readFile(userPath); ok {
			m.UserFile = &fm
		}
	}

	return m
}

func readFile(path string) (FileMemory, bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return FileMemory{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return FileMemory{}, false
	}
	return FileMemory{Path: path, Content: string(data)}, true
}

func (m *Memory) Combined() string {
	if m == nil || (m.UserFile == nil && len(m.ProjectFiles) == 0) {
		return ""
	}

	var b strings.Builder

	if m.UserFile != nil {
		b.WriteString("# User memory (")
		b.WriteString(m.UserFile.Path)
		b.WriteString(")\n")
		b.WriteString(m.UserFile.Content)
		if !strings.HasSuffix(m.UserFile.Content, "\n") {
			b.WriteString("\n")
		}
	}

	for i, f := range m.ProjectFiles {
		if b.Len() > 0 && (i > 0 || m.UserFile != nil) {
			b.WriteString("\n")
		}
		b.WriteString("# Project memory (")
		b.WriteString(f.Path)
		b.WriteString(")\n")
		b.WriteString(f.Content)
		if !strings.HasSuffix(f.Content, "\n") {
			b.WriteString("\n")
		}
	}

	return b.String()
}
