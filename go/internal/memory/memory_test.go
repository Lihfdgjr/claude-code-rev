package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withFakeHome sets HOME (and on Windows USERPROFILE) to a temp dir so
// LoadProject can't find a user CLAUDE.md from the real home directory.
func withFakeHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	return tmp
}

func TestLoadProjectEmptyWhenNoClaudeMD(t *testing.T) {
	withFakeHome(t)
	dir := t.TempDir()
	m := LoadProject(dir)
	if m == nil {
		t.Fatal("LoadProject returned nil")
	}
	if len(m.ProjectFiles) != 0 {
		t.Errorf("expected no project files, got %d", len(m.ProjectFiles))
	}
	if m.UserFile != nil {
		t.Errorf("expected no user file, got %+v", m.UserFile)
	}
}

func TestCombinedEmptyMemory(t *testing.T) {
	m := &Memory{}
	if got := m.Combined(); got != "" {
		t.Errorf("Combined() = %q, want empty", got)
	}
}

func TestCombinedNilMemory(t *testing.T) {
	var m *Memory
	if got := m.Combined(); got != "" {
		t.Errorf("nil Combined() = %q, want empty", got)
	}
}

func TestLoadProjectFindsClaudeMD(t *testing.T) {
	withFakeHome(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(path, []byte("project rules"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := LoadProject(dir)
	if len(m.ProjectFiles) != 1 {
		t.Fatalf("expected 1 project file, got %d", len(m.ProjectFiles))
	}
	if m.ProjectFiles[0].Content != "project rules" {
		t.Errorf("project file content = %q", m.ProjectFiles[0].Content)
	}
}

func TestCombinedIncludesProjectFile(t *testing.T) {
	m := &Memory{
		ProjectFiles: []FileMemory{{Path: "/p/CLAUDE.md", Content: "rules"}},
	}
	got := m.Combined()
	if !strings.Contains(got, "Project memory") {
		t.Errorf("expected 'Project memory' header, got %q", got)
	}
	if !strings.Contains(got, "rules") {
		t.Errorf("expected 'rules' content, got %q", got)
	}
}
