package skills

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type Skill struct {
	Name        string
	Description string
	Path        string
	Body        string
}

type Loader struct {
	projectDir string
}

func New(projectDir string) *Loader {
	return &Loader{projectDir: projectDir}
}

func (l *Loader) Load() ([]*Skill, error) {
	byName := make(map[string]*Skill)
	var order []string

	add := func(s *Skill, override bool) {
		if s == nil || s.Name == "" {
			return
		}
		if _, ok := byName[s.Name]; !ok {
			order = append(order, s.Name)
		} else if !override {
			return
		}
		byName[s.Name] = s
	}

	if home, err := os.UserHomeDir(); err == nil {
		userDir := filepath.Join(home, ".claude", "skills")
		userSkills, err := loadDir(userDir)
		if err != nil {
			return nil, err
		}
		for _, s := range userSkills {
			add(s, false)
		}
	}

	if l.projectDir != "" {
		projDir := filepath.Join(l.projectDir, ".claude", "skills")
		projSkills, err := loadDir(projDir)
		if err != nil {
			return nil, err
		}
		for _, s := range projSkills {
			add(s, true)
		}
	}

	out := make([]*Skill, 0, len(order))
	for _, name := range order {
		out = append(out, byName[name])
	}
	return out, nil
}

func loadDir(dir string) ([]*Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var skills []*Skill
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())
		if e.IsDir() {
			skillPath := filepath.Join(full, "SKILL.md")
			if _, err := os.Stat(skillPath); err == nil {
				s, err := parseSkill(skillPath, e.Name())
				if err != nil {
					return nil, err
				}
				if s != nil {
					skills = append(skills, s)
				}
			}
			continue
		}
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		s, err := parseSkill(full, base)
		if err != nil {
			return nil, err
		}
		if s != nil {
			skills = append(skills, s)
		}
	}
	return skills, nil
}

func parseSkill(path, defaultName string) (*Skill, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	meta := map[string]string{}
	var bodyLines []string
	state := "start"

	for scanner.Scan() {
		line := scanner.Text()
		switch state {
		case "start":
			if strings.TrimSpace(line) == "---" {
				state = "front"
				continue
			}
			state = "body"
			bodyLines = append(bodyLines, line)
		case "front":
			trimmed := strings.TrimSpace(line)
			if trimmed == "---" || trimmed == "" {
				state = "body"
				continue
			}
			if idx := strings.Index(line, ":"); idx > 0 {
				key := strings.TrimSpace(line[:idx])
				val := strings.TrimSpace(line[idx+1:])
				val = strings.Trim(val, "\"'")
				if key != "" {
					meta[strings.ToLower(key)] = val
				}
			}
		case "body":
			bodyLines = append(bodyLines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	name := meta["name"]
	if name == "" {
		name = defaultName
	}

	body := strings.Join(bodyLines, "\n")
	body = strings.TrimLeft(body, "\n")

	return &Skill{
		Name:        name,
		Description: meta["description"],
		Path:        path,
		Body:        body,
	}, nil
}
