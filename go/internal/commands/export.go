package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"claudecode/internal/core"
)

type exportCmd struct{}

// NewExport returns a /export command that writes the current transcript to markdown.
func NewExport() core.Command {
	return &exportCmd{}
}

func (c *exportCmd) Name() string     { return "export" }
func (c *exportCmd) Synopsis() string { return "Export the current conversation as a markdown transcript" }

func (c *exportCmd) Run(ctx context.Context, args string, sess core.Session) error {
	path := strings.TrimSpace(args)
	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			sess.Notify(core.NotifyError, fmt.Sprintf("export: %v", err))
			return err
		}
		stamp := time.Now().UTC().Format("2006-01-02T15-04-05")
		path = filepath.Join(cwd, fmt.Sprintf("transcript-%s.md", stamp))
	}

	md := renderMarkdown(sess.History())
	if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("export: %v", err))
		return err
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("exported transcript to %s", path))
	return nil
}

func renderMarkdown(msgs []core.Message) string {
	var b strings.Builder
	b.WriteString("# Conversation Transcript\n\n")
	for _, m := range msgs {
		switch m.Role {
		case core.RoleUser:
			b.WriteString("## User\n\n")
			for _, blk := range m.Blocks {
				writeBlockMarkdown(&b, blk)
			}
		case core.RoleAssistant:
			b.WriteString("## Assistant\n\n")
			for _, blk := range m.Blocks {
				writeBlockMarkdown(&b, blk)
			}
		case core.RoleSystem:
			b.WriteString("## System\n\n")
			for _, blk := range m.Blocks {
				writeBlockMarkdown(&b, blk)
			}
		default:
			fmt.Fprintf(&b, "## %s\n\n", string(m.Role))
			for _, blk := range m.Blocks {
				writeBlockMarkdown(&b, blk)
			}
		}
	}
	return b.String()
}

func writeBlockMarkdown(b *strings.Builder, blk core.Block) {
	switch v := blk.(type) {
	case core.TextBlock:
		text := strings.TrimRight(v.Text, "\n")
		if text == "" {
			return
		}
		b.WriteString(text)
		b.WriteString("\n\n")
	case core.ToolUseBlock:
		fmt.Fprintf(b, "**Tool call: `%s`**\n\n", v.Name)
		input := string(v.Input)
		if input == "" {
			input = "{}"
		}
		b.WriteString("```json\n")
		b.WriteString(input)
		if !strings.HasSuffix(input, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
	case core.ToolResultBlock:
		header := "**Tool result**"
		if v.IsError {
			header = "**Tool result (error)**"
		}
		fmt.Fprintf(b, "%s\n\n", header)
		b.WriteString("```\n")
		b.WriteString(v.Content)
		if !strings.HasSuffix(v.Content, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
	case core.ThinkingBlock:
		text := strings.TrimRight(v.Text, "\n")
		if text == "" {
			return
		}
		b.WriteString("> _thinking_\n>\n")
		for _, line := range strings.Split(text, "\n") {
			b.WriteString("> ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
}
