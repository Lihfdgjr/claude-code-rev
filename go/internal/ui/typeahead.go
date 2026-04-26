package ui

import (
	"sort"
	"strings"

	"claudecode/internal/core"
)

const typeaheadMax = 8

// Suggestion is one autocomplete row.
type Suggestion struct {
	Name     string
	Synopsis string
}

// Typeahead computes slash-command suggestions for the prompt input. It is
// stateful only to cache the last query; matching itself is recomputed on
// every Update call.
type Typeahead struct {
	reg      core.CommandRegistry
	last     string
	matches  []Suggestion
	selected int
}

// NewTypeahead binds a typeahead helper to a command registry.
func NewTypeahead(reg core.CommandRegistry) *Typeahead {
	return &Typeahead{reg: reg}
}

// Suggestions returns the most recent suggestion list without recomputing.
func (t *Typeahead) Suggestions() []Suggestion {
	if t == nil {
		return nil
	}
	return t.matches
}

// Selected returns the currently selected suggestion index. It clamps to the
// valid range; if there are no matches it returns 0.
func (t *Typeahead) Selected() int {
	if t == nil || len(t.matches) == 0 {
		return 0
	}
	if t.selected >= len(t.matches) {
		t.selected = len(t.matches) - 1
	}
	if t.selected < 0 {
		t.selected = 0
	}
	return t.selected
}

// SetSelected updates the selection cursor with wrap-around. Useful for Tab
// cycling from the host model.
func (t *Typeahead) SetSelected(i int) {
	if t == nil || len(t.matches) == 0 {
		return
	}
	n := len(t.matches)
	t.selected = ((i % n) + n) % n
}

// Cycle advances the selection by delta with wrap-around.
func (t *Typeahead) Cycle(delta int) {
	if t == nil || len(t.matches) == 0 {
		return
	}
	t.SetSelected(t.Selected() + delta)
}

// Update recomputes suggestions for the given input. When input does not
// start with '/' the returned slice is empty. Matching is case-insensitive
// substring against command names; results are capped at typeaheadMax.
func (t *Typeahead) Update(input string) []Suggestion {
	if t == nil {
		return nil
	}
	if !strings.HasPrefix(input, "/") {
		t.matches = nil
		t.last = input
		t.selected = 0
		return nil
	}
	if input == t.last && t.matches != nil {
		return t.matches
	}
	t.last = input

	q := strings.ToLower(strings.TrimPrefix(input, "/"))
	// Drop anything past the first whitespace; we only complete the verb.
	if idx := strings.IndexAny(q, " \t"); idx >= 0 {
		q = q[:idx]
	}

	if t.reg == nil {
		t.matches = nil
		return nil
	}

	all := t.reg.All()
	var out []Suggestion
	for _, c := range all {
		name := strings.ToLower(c.Name())
		if q == "" || strings.Contains(name, q) {
			out = append(out, Suggestion{Name: c.Name(), Synopsis: c.Synopsis()})
		}
	}

	// Prefer prefix matches over interior matches, then alphabetical.
	sort.SliceStable(out, func(i, j int) bool {
		ai := strings.HasPrefix(strings.ToLower(out[i].Name), q)
		aj := strings.HasPrefix(strings.ToLower(out[j].Name), q)
		if ai != aj {
			return ai
		}
		return out[i].Name < out[j].Name
	})

	if len(out) > typeaheadMax {
		out = out[:typeaheadMax]
	}
	t.matches = out
	if t.selected >= len(out) {
		t.selected = 0
	}
	return out
}

// CompleteValue returns the input string with the verb replaced by the
// currently selected suggestion (and a trailing space). If there is no
// suggestion or the input is not a slash command, the input is returned
// unchanged.
func (t *Typeahead) CompleteValue(input string) string {
	if t == nil || len(t.matches) == 0 || !strings.HasPrefix(input, "/") {
		return input
	}
	sel := t.matches[t.Selected()]

	rest := ""
	body := strings.TrimPrefix(input, "/")
	if idx := strings.IndexAny(body, " \t"); idx >= 0 {
		rest = body[idx:]
	}
	return "/" + sel.Name + rest
}

// View renders the suggestion box that should sit above the prompt. Returns
// an empty string when there is nothing to show.
func (t *Typeahead) View(width int) string {
	if t == nil || len(t.matches) == 0 {
		return ""
	}
	w := width
	if w > 70 {
		w = 70
	}
	if w < 20 {
		w = 20
	}

	var b strings.Builder
	for i, s := range t.matches {
		row := s.Name
		if s.Synopsis != "" {
			row = padRight(s.Name, 18) + "  " + s.Synopsis
		}
		// Clip to width budget.
		if len(row) > w-4 {
			row = row[:w-4]
		}
		if i == t.Selected() {
			b.WriteString(typeaheadSelectedStyle.Render("> " + row))
		} else {
			b.WriteString(typeaheadItemStyle.Render("  " + row))
		}
		if i < len(t.matches)-1 {
			b.WriteString("\n")
		}
	}
	return typeaheadBoxStyle.Width(w).Render(b.String())
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
