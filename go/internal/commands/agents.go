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

type agentsCmd struct{}

func NewAgents() core.Command { return &agentsCmd{} }

func (agentsCmd) Name() string     { return "agents" }
func (agentsCmd) Synopsis() string { return "Manage sub-agent definitions" }

func (agentsCmd) Run(ctx context.Context, args string, sess core.Session) error {
	fields := strings.Fields(strings.TrimSpace(args))
	if len(fields) == 0 {
		return agentsList(sess)
	}
	switch fields[0] {
	case "create", "new":
		if len(fields) < 2 {
			sess.Notify(core.NotifyError, "usage: /agents create <name>")
			return nil
		}
		return agentsCreate(sess, fields[1])
	case "show":
		if len(fields) < 2 {
			sess.Notify(core.NotifyError, "usage: /agents show <name>")
			return nil
		}
		return agentsShow(sess, fields[1])
	case "edit":
		if len(fields) < 2 {
			sess.Notify(core.NotifyError, "usage: /agents edit <name>")
			return nil
		}
		return agentsEdit(sess, fields[1])
	default:
		sess.Notify(core.NotifyError, fmt.Sprintf("unknown subcommand: %s", fields[0]))
		return nil
	}
}

func agentDirs() []string {
	var dirs []string
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, ".claude", "agents"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".claude", "agents"))
	}
	return dirs
}

func agentsList(sess core.Session) error {
	type entry struct {
		name string
		desc string
		path string
	}
	seen := map[string]entry{}
	var order []string
	for _, dir := range agentDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			name, desc := readAgentMeta(path)
			if name == "" {
				name = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = entry{name: name, desc: desc, path: path}
			order = append(order, name)
		}
	}
	if len(order) == 0 {
		sess.Notify(core.NotifyInfo, "No agent definitions found in .claude/agents/.")
		return nil
	}
	sort.Strings(order)
	var b strings.Builder
	b.WriteString("Agents:\n")
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

func agentsCreate(sess core.Session, name string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	dir := filepath.Join(cwd, ".claude", "agents")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, name+".md")
	if _, err := os.Stat(path); err == nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("agent already exists: %s", path))
		return nil
	}
	tpl := fmt.Sprintf(`---
name: %s
description: Short description of what this agent does.
tools: Read, Grep, Glob, Bash
model: sonnet
---

You are %s. Describe the agent's role, responsibilities, and any
constraints here. The body of the file becomes the system prompt.
`, name, name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(tpl), 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Created agent at %s", path))
	return nil
}

func agentsShow(sess core.Session, name string) error {
	path := findAgentFile(name)
	if path == "" {
		sess.Notify(core.NotifyError, fmt.Sprintf("agent %q not found", name))
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	front, body := splitFrontmatter(string(data))
	var b strings.Builder
	fmt.Fprintf(&b, "Agent: %s\nPath: %s\n", name, path)
	if front != "" {
		b.WriteString("---\n")
		b.WriteString(front)
		if !strings.HasSuffix(front, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("---\n")
	}
	body = strings.TrimSpace(body)
	if len(body) > 500 {
		body = body[:500] + "..."
	}
	b.WriteString(body)
	sess.Notify(core.NotifyInfo, b.String())
	return nil
}

func agentsEdit(sess core.Session, name string) error {
	path := findAgentFile(name)
	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		path = filepath.Join(cwd, ".claude", "agents", name+".md")
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Open %s in your editor.", path))
	return nil
}

func findAgentFile(name string) string {
	for _, dir := range agentDirs() {
		path := filepath.Join(dir, name+".md")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func readAgentMeta(path string) (name, desc string) {
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

func splitFrontmatter(s string) (front, body string) {
	s = strings.TrimLeft(s, "")
	if !strings.HasPrefix(s, "---") {
		return "", s
	}
	rest := s[3:]
	if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	} else if strings.HasPrefix(rest, "\r\n") {
		rest = rest[2:]
	} else {
		return "", s
	}
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", s
	}
	front = rest[:end]
	body = rest[end+4:]
	body = strings.TrimLeft(body, "\r\n")
	return front, body
}
