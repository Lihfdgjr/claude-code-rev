package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type rawInner struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

type rawMatcherEntry struct {
	Matcher string     `json:"matcher"`
	Hooks   []rawInner `json:"hooks"`
}

type rawSettings struct {
	Hooks map[string][]rawMatcherEntry `json:"hooks"`
}

func LoadFromSettings(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return nil, err
	}

	var s rawSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	cfg := Config{}
	for name, entries := range s.Hooks {
		ev := EventName(name)
		var specs []HookSpec
		for _, entry := range entries {
			for _, h := range entry.Hooks {
				specs = append(specs, HookSpec{
					Matcher: entry.Matcher,
					Type:    h.Type,
					Command: h.Command,
					Timeout: h.Timeout,
				})
			}
		}
		if len(specs) > 0 {
			cfg[ev] = specs
		}
	}
	return cfg, nil
}
