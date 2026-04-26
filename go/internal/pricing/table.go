package pricing

import (
	"fmt"

	"claudecode/internal/core"
)

// Pricing is in dollars per million tokens.
type Pricing struct {
	Input      float64
	Output     float64
	CacheRead  float64
	CacheWrite float64
}

var Table = map[string]Pricing{
	"claude-opus-4-7": {
		Input:      15.0,
		Output:     75.0,
		CacheRead:  1.50,
		CacheWrite: 18.75,
	},
	"claude-opus-4-5": {
		Input:      15.0,
		Output:     75.0,
		CacheRead:  1.50,
		CacheWrite: 18.75,
	},
	"claude-sonnet-4-6": {
		Input:      3.0,
		Output:     15.0,
		CacheRead:  0.30,
		CacheWrite: 3.75,
	},
	"claude-haiku-4-5-20251001": {
		Input:      0.80,
		Output:     4.0,
		CacheRead:  0.08,
		CacheWrite: 1.0,
	},
}

func lookup(model string) Pricing {
	if p, ok := Table[model]; ok {
		return p
	}
	return Table["claude-sonnet-4-6"]
}

// Estimate returns the total dollar cost for the given usage.
func Estimate(model string, u core.Usage) float64 {
	p := lookup(model)
	const m = 1_000_000.0
	return float64(u.InputTokens)*p.Input/m +
		float64(u.OutputTokens)*p.Output/m +
		float64(u.CacheReadTokens)*p.CacheRead/m +
		float64(u.CacheCreationTokens)*p.CacheWrite/m
}

// FormatUSD renders a dollar amount with 4 decimals.
func FormatUSD(d float64) string {
	return fmt.Sprintf("$%.4f", d)
}
