package watcher

import (
	"encoding/json"
	"os"
	"strings"
)

func WatchSettings(w *Watcher, settingsPath string, onChange func(map[string]interface{})) {
	if w == nil || settingsPath == "" || onChange == nil {
		return
	}
	w.Add(settingsPath, func(path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			onChange(nil)
			return
		}
		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			return
		}
		onChange(m)
	})
}

func WatchCLAUDEmd(w *Watcher, paths []string, onChange func(combined string)) {
	if w == nil || onChange == nil || len(paths) == 0 {
		return
	}
	rebuild := func(_ string) {
		var b strings.Builder
		for i, p := range paths {
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			if i == 0 {
				b.WriteString("# User memory (")
			} else {
				b.WriteString("# Project memory (")
			}
			b.WriteString(p)
			b.WriteString(")\n")
			b.Write(data)
			if len(data) == 0 || data[len(data)-1] != '\n' {
				b.WriteString("\n")
			}
		}
		onChange(b.String())
	}
	for _, p := range paths {
		w.Add(p, rebuild)
	}
}
