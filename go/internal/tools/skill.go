package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"claudecode/internal/core"
	"claudecode/internal/skills"
)

type skillTool struct {
	loader *skills.Loader

	once   sync.Once
	cached []*skills.Skill
	loadErr error
}

type skillInput struct {
	Skill string `json:"skill"`
	Args  string `json:"args,omitempty"`
}

func NewSkill(loader *skills.Loader) core.Tool {
	return &skillTool{loader: loader}
}

func (s *skillTool) Name() string { return "Skill" }

func (s *skillTool) Description() string {
	return "Invoke a named skill, returning its instructions. Optional args are substituted into the skill body where it contains the literal token {{args}}."
}

func (s *skillTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "skill": {"type": "string", "description": "Name of the skill to invoke"},
    "args": {"type": "string", "description": "Optional argument string"}
  },
  "required": ["skill"],
  "additionalProperties": false
}`)
}

func (s *skillTool) load() ([]*skills.Skill, error) {
	s.once.Do(func() {
		if s.loader == nil {
			s.loadErr = fmt.Errorf("skill loader not configured")
			return
		}
		s.cached, s.loadErr = s.loader.Load()
	})
	return s.cached, s.loadErr
}

func (s *skillTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in skillInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if strings.TrimSpace(in.Skill) == "" {
		return "", fmt.Errorf("skill is required")
	}

	all, err := s.load()
	if err != nil {
		return "", err
	}

	var match *skills.Skill
	for _, sk := range all {
		if sk.Name == in.Skill {
			match = sk
			break
		}
	}
	if match == nil {
		names := make([]string, 0, len(all))
		for _, sk := range all {
			names = append(names, sk.Name)
		}
		sort.Strings(names)
		if len(names) == 0 {
			return "", fmt.Errorf("skill %q not found (no skills available)", in.Skill)
		}
		return "", fmt.Errorf("skill %q not found; available: %s", in.Skill, strings.Join(names, ", "))
	}

	body := match.Body
	if in.Args != "" && strings.Contains(body, "{{args}}") {
		body = strings.ReplaceAll(body, "{{args}}", in.Args)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Skill: %s\n", match.Name)
	if match.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", match.Description)
	}
	b.WriteString("\n")
	b.WriteString(body)
	return strings.TrimRight(b.String(), "\n"), nil
}
