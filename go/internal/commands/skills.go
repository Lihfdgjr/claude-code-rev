package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"claudecode/internal/core"
)

type skillsCmd struct{}

func NewSkills() core.Command { return &skillsCmd{} }

func (skillsCmd) Name() string     { return "skills" }
func (skillsCmd) Synopsis() string { return "Manage skills" }

func (skillsCmd) Run(ctx context.Context, args string, sess core.Session) error {
	fields := strings.Fields(strings.TrimSpace(args))
	if len(fields) == 0 {
		return skillsList(sess)
	}
	switch fields[0] {
	case "new", "create":
		if len(fields) < 2 {
			sess.Notify(core.NotifyError, "usage: /skills new <name>")
			return nil
		}
		return skillsNew(sess, fields[1])
	case "show":
		if len(fields) < 2 {
			sess.Notify(core.NotifyError, "usage: /skills show <name>")
			return nil
		}
		return skillsShow(sess, fields[1])
	case "install":
		if len(fields) < 2 {
			sess.Notify(core.NotifyError, "usage: /skills install <git-url>")
			return nil
		}
		return skillsInstall(sess, fields[1])
	default:
		sess.Notify(core.NotifyError, fmt.Sprintf("unknown subcommand: %s", fields[0]))
		return nil
	}
}

func skillDirs() []string {
	var dirs []string
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, ".claude", "skills"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".claude", "skills"))
	}
	return dirs
}

func skillsList(sess core.Session) error {
	type entry struct {
		name string
		desc string
		path string
	}
	seen := map[string]entry{}
	var order []string
	for _, dir := range skillDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			path := filepath.Join(dir, e.Name(), "SKILL.md")
			if _, err := os.Stat(path); err != nil {
				continue
			}
			name, desc := readSkillMeta(path)
			if name == "" {
				name = e.Name()
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = entry{name: name, desc: desc, path: path}
			order = append(order, name)
		}
	}
	if len(order) == 0 {
		sess.Notify(core.NotifyInfo, "No skills found in .claude/skills/.")
		return nil
	}
	sort.Strings(order)
	var b strings.Builder
	b.WriteString("Skills:\n")
	for _, n := range order {
		e := seen[n]
		if e.desc != "" {
			fmt.Fprintf(&b, "  %s - %s\n", e.name, e.desc)
		} else {
			fmt.Fprintf(&b, "  %s\n", e.name)
		}
	}
	sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
	return nil
}

func skillsNew(sess core.Session, name string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	dir := filepath.Join(cwd, ".claude", "skills", name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, "SKILL.md")
	if _, err := os.Stat(path); err == nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("skill already exists: %s", path))
		return nil
	}
	tpl := fmt.Sprintf(`---
name: %s
description: Short description of when to invoke this skill.
---

# %s

Describe the skill's behavior, inputs, and outputs here. The body of
this file is shown to the model as guidance when the skill is invoked.
`, name, name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(tpl), 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Created skill at %s", path))
	return nil
}

func skillsShow(sess core.Session, name string) error {
	for _, dir := range skillDirs() {
		path := filepath.Join(dir, name, "SKILL.md")
		if _, err := os.Stat(path); err != nil {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sess.Notify(core.NotifyInfo, fmt.Sprintf("Skill: %s\nPath: %s\n\n%s", name, path, string(data)))
		return nil
	}
	sess.Notify(core.NotifyError, fmt.Sprintf("skill %q not found", name))
	return nil
}

func skillsInstall(sess core.Session, url string) error {
	name := skillNameFromURL(url)
	sess.Notify(core.NotifyInfo,
		fmt.Sprintf("Manual install: clone %s into ~/.claude/skills/%s/.", url, name))
	return nil
}

func skillNameFromURL(url string) string {
	u := strings.TrimSuffix(url, "/")
	u = strings.TrimSuffix(u, ".git")
	if i := strings.LastIndex(u, "/"); i >= 0 {
		u = u[i+1:]
	}
	if u == "" {
		return "<name>"
	}
	return u
}

func readSkillMeta(path string) (name, desc string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	front, _ := splitFrontmatter(string(data))
	if front == "" {
		return "", ""
	}
	for _, line := range strings.Split(front, "\n") {
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(line[:idx]))
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, `"'`)
		switch key {
		case "name":
			name = val
		case "description":
			desc = val
		}
	}
	return name, desc
}
