package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type textDiffTool struct{}

type textDiffInput struct {
	A string `json:"a"`
	B string `json:"b"`
}

func NewTextDiff() core.Tool { return &textDiffTool{} }

func (textDiffTool) Name() string { return "TextDiff" }

func (textDiffTool) Description() string {
	return "Compute a unified line-level diff between two strings using LCS. Returns patch text with @@ hunks."
}

func (textDiffTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "a": {"type": "string"},
    "b": {"type": "string"}
  },
  "required": ["a", "b"],
  "additionalProperties": false
}`)
}

func (textDiffTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in textDiffInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	aLines := splitDiffLines(in.A)
	bLines := splitDiffLines(in.B)
	ops := lcsDiff(aLines, bLines)
	return renderUnifiedDiff(ops), nil
}

func splitDiffLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// diffOp: ' ' equal, '-' delete from a, '+' add from b
type diffOp struct {
	kind byte
	line string
}

func lcsDiff(a, b []string) []diffOp {
	n, m := len(a), len(b)
	// dp[i][j] = LCS length of a[i:] and b[j:]
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var out []diffOp
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			out = append(out, diffOp{' ', a[i]})
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			out = append(out, diffOp{'-', a[i]})
			i++
		} else {
			out = append(out, diffOp{'+', b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, diffOp{'-', a[i]})
	}
	for ; j < m; j++ {
		out = append(out, diffOp{'+', b[j]})
	}
	return out
}

func renderUnifiedDiff(ops []diffOp) string {
	const ctxN = 3
	type hunk struct {
		aStart, bStart int
		lines          []diffOp
	}

	// First, walk ops to assign per-op a/b line numbers (1-based).
	type stamped struct {
		op   diffOp
		aLn  int
		bLn  int
	}
	stamps := make([]stamped, 0, len(ops))
	ai, bi := 1, 1
	for _, o := range ops {
		s := stamped{op: o, aLn: ai, bLn: bi}
		switch o.kind {
		case ' ':
			ai++
			bi++
		case '-':
			ai++
		case '+':
			bi++
		}
		stamps = append(stamps, s)
	}

	// Group changes into hunks with up to ctxN context lines around each change cluster.
	var hunks []hunk
	i := 0
	for i < len(stamps) {
		if stamps[i].op.kind == ' ' {
			i++
			continue
		}
		// start hunk: include up to ctxN preceding context lines
		start := i
		for start > 0 && i-start < ctxN && stamps[start-1].op.kind == ' ' {
			start--
		}
		// extend hunk: include change runs and short equal runs (<= 2*ctxN gap merges)
		end := i
		for end < len(stamps) {
			if stamps[end].op.kind != ' ' {
				end++
				continue
			}
			// look ahead for more changes within 2*ctxN equal lines
			gap := 0
			k := end
			for k < len(stamps) && stamps[k].op.kind == ' ' && gap < 2*ctxN {
				gap++
				k++
			}
			if k < len(stamps) && stamps[k].op.kind != ' ' {
				end = k
				continue
			}
			break
		}
		// trailing context: include up to ctxN equal lines after
		tail := 0
		for end < len(stamps) && tail < ctxN && stamps[end].op.kind == ' ' {
			end++
			tail++
		}
		h := hunk{aStart: stamps[start].aLn, bStart: stamps[start].bLn}
		for k := start; k < end; k++ {
			h.lines = append(h.lines, stamps[k].op)
		}
		hunks = append(hunks, h)
		i = end
	}

	if len(hunks) == 0 {
		return ""
	}

	var b2 strings.Builder
	b2.WriteString("--- a\n+++ b\n")
	for _, h := range hunks {
		var aCount, bCount int
		for _, l := range h.lines {
			switch l.kind {
			case ' ':
				aCount++
				bCount++
			case '-':
				aCount++
			case '+':
				bCount++
			}
		}
		aStart := h.aStart
		if aCount == 0 {
			aStart = h.aStart - 1
			if aStart < 0 {
				aStart = 0
			}
		}
		bStart := h.bStart
		if bCount == 0 {
			bStart = h.bStart - 1
			if bStart < 0 {
				bStart = 0
			}
		}
		fmt.Fprintf(&b2, "@@ -%d,%d +%d,%d @@\n", aStart, aCount, bStart, bCount)
		for _, l := range h.lines {
			b2.WriteByte(l.kind)
			b2.WriteString(l.line)
			b2.WriteByte('\n')
		}
	}
	return b2.String()
}
