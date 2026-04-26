package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"claudecode/internal/core"
)

type toolSearchTool struct {
	reg core.ToolRegistry
}

func NewToolSearch(reg core.ToolRegistry) core.Tool { return &toolSearchTool{reg: reg} }

func (t *toolSearchTool) Name() string { return "ToolSearch" }

func (t *toolSearchTool) Description() string {
	return "Look up tool schemas. Use 'select:Foo,Bar' for exact-name lookup, or pass keywords to rank by name-prefix match and substring presence."
}

func (t *toolSearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {"type": "string"},
			"max_results": {"type": "integer", "minimum": 1, "maximum": 50}
		},
		"required": ["query"],
		"additionalProperties": false
	}`)
}

func truncateSchema(raw json.RawMessage, max int) string {
	s := string(raw)
	if len(s) <= max {
		return s
	}
	return s[:max] + "...[truncated]"
}

func formatTool(tool core.Tool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\n", tool.Name())
	desc := tool.Description()
	if desc == "" {
		desc = "(no description)"
	}
	fmt.Fprintf(&b, "description: %s\n", desc)
	fmt.Fprintf(&b, "schema: %s\n", truncateSchema(tool.Schema(), 500))
	return b.String()
}

func scoreTool(tool core.Tool, terms []string) int {
	if len(terms) == 0 {
		return 0
	}
	name := strings.ToLower(tool.Name())
	desc := strings.ToLower(tool.Description())
	score := 0
	for _, term := range terms {
		if term == "" {
			continue
		}
		if strings.HasPrefix(name, term) {
			score += 100
		}
		if strings.Contains(name, term) {
			score += 25
		}
		if strings.Contains(desc, term) {
			score += 5
		}
	}
	return score
}

func (t *toolSearchTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Query      string `json:"query"`
		MaxResults int    `json:"max_results"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if strings.TrimSpace(args.Query) == "" {
		return "", fmt.Errorf("query required")
	}
	if t.reg == nil {
		return "", fmt.Errorf("registry not configured")
	}
	max := args.MaxResults
	if max <= 0 {
		max = 5
	}
	all := t.reg.All()

	if strings.HasPrefix(args.Query, "select:") {
		names := strings.Split(strings.TrimPrefix(args.Query, "select:"), ",")
		var found []core.Tool
		var missing []string
		for _, n := range names {
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			if tool, ok := t.reg.Get(n); ok {
				found = append(found, tool)
			} else {
				missing = append(missing, n)
			}
		}
		var b strings.Builder
		if len(found) == 0 {
			fmt.Fprintf(&b, "No tools matched select query.\n")
		} else {
			fmt.Fprintf(&b, "Matched %d tool(s):\n\n", len(found))
			for _, tool := range found {
				b.WriteString(formatTool(tool))
				b.WriteString("\n")
			}
		}
		if len(missing) > 0 {
			fmt.Fprintf(&b, "Unknown: %s\n", strings.Join(missing, ", "))
		}
		return strings.TrimRight(b.String(), "\n") + "\n", nil
	}

	terms := strings.Fields(strings.ToLower(args.Query))
	type scored struct {
		tool  core.Tool
		score int
		idx   int
	}
	var ranked []scored
	for i, tool := range all {
		s := scoreTool(tool, terms)
		if s > 0 {
			ranked = append(ranked, scored{tool, s, i})
		}
	}
	if len(ranked) == 0 {
		return fmt.Sprintf("No tools matched query %q (registry has %d tools).\n", args.Query, len(all)), nil
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].idx < ranked[j].idx
	})
	if len(ranked) > max {
		ranked = ranked[:max]
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Top %d match(es) for %q:\n\n", len(ranked), args.Query)
	for _, r := range ranked {
		b.WriteString(formatTool(r.tool))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n", nil
}
