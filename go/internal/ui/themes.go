package ui

import (
	"fmt"
	"sort"
	"sync/atomic"

	"github.com/charmbracelet/lipgloss"
)

// Theme is a named palette applied to the UI styles. Colors map to logical
// roles rather than concrete style names so the palette stays compact.
type Theme struct {
	Name        string
	Description string
	UserPrefix  lipgloss.Color
	UserText    lipgloss.Color
	Assistant   lipgloss.Color
	Thinking    lipgloss.Color
	Tool        lipgloss.Color
	Error       lipgloss.Color
	Status      lipgloss.Color
	Border      lipgloss.Color
}

// Themes is the registry of built-in palettes, keyed by lower-case name.
var Themes = map[string]Theme{
	"dark": {
		Name:        "dark",
		Description: "Default dark theme with cyan accents",
		UserPrefix:  lipgloss.Color("14"),
		UserText:    lipgloss.Color("15"),
		Assistant:   lipgloss.Color("252"),
		Thinking:    lipgloss.Color("244"),
		Tool:        lipgloss.Color("111"),
		Error:       lipgloss.Color("9"),
		Status:      lipgloss.Color("117"),
		Border:      lipgloss.Color("117"),
	},
	"light": {
		Name:        "light",
		Description: "Light terminal palette",
		UserPrefix:  lipgloss.Color("4"),
		UserText:    lipgloss.Color("0"),
		Assistant:   lipgloss.Color("236"),
		Thinking:    lipgloss.Color("241"),
		Tool:        lipgloss.Color("27"),
		Error:       lipgloss.Color("1"),
		Status:      lipgloss.Color("25"),
		Border:      lipgloss.Color("25"),
	},
	"solarized": {
		Name:        "solarized",
		Description: "Solarized-inspired warm palette",
		UserPrefix:  lipgloss.Color("#268bd2"),
		UserText:    lipgloss.Color("#eee8d5"),
		Assistant:   lipgloss.Color("#93a1a1"),
		Thinking:    lipgloss.Color("#586e75"),
		Tool:        lipgloss.Color("#2aa198"),
		Error:       lipgloss.Color("#dc322f"),
		Status:      lipgloss.Color("#b58900"),
		Border:      lipgloss.Color("#073642"),
	},
	"high-contrast": {
		Name:        "high-contrast",
		Description: "Maximum-contrast palette for visibility",
		UserPrefix:  lipgloss.Color("15"),
		UserText:    lipgloss.Color("15"),
		Assistant:   lipgloss.Color("15"),
		Thinking:    lipgloss.Color("11"),
		Tool:        lipgloss.Color("14"),
		Error:       lipgloss.Color("9"),
		Status:      lipgloss.Color("15"),
		Border:      lipgloss.Color("15"),
	},
	"colorblind": {
		Name:        "colorblind",
		Description: "Blue/orange palette safe for red-green colorblindness",
		UserPrefix:  lipgloss.Color("39"),
		UserText:    lipgloss.Color("231"),
		Assistant:   lipgloss.Color("252"),
		Thinking:    lipgloss.Color("244"),
		Tool:        lipgloss.Color("214"),
		Error:       lipgloss.Color("208"),
		Status:      lipgloss.Color("39"),
		Border:      lipgloss.Color("39"),
	},
	"monokai": {
		Name:        "monokai",
		Description: "Classic Monokai: black bg, magenta keywords, green strings",
		UserPrefix:  lipgloss.Color("#f92672"),
		UserText:    lipgloss.Color("#f8f8f2"),
		Assistant:   lipgloss.Color("#a6e22e"),
		Thinking:    lipgloss.Color("#75715e"),
		Tool:        lipgloss.Color("#e6db74"),
		Error:       lipgloss.Color("#f92672"),
		Status:      lipgloss.Color("#66d9ef"),
		Border:      lipgloss.Color("#49483e"),
	},
	"dracula": {
		Name:        "dracula",
		Description: "Dracula: dark purple bg, cyan accents, pink highlights",
		UserPrefix:  lipgloss.Color("#ff79c6"),
		UserText:    lipgloss.Color("#f8f8f2"),
		Assistant:   lipgloss.Color("#8be9fd"),
		Thinking:    lipgloss.Color("#6272a4"),
		Tool:        lipgloss.Color("#50fa7b"),
		Error:       lipgloss.Color("#ff5555"),
		Status:      lipgloss.Color("#bd93f9"),
		Border:      lipgloss.Color("#44475a"),
	},
	"gruvbox": {
		Name:        "gruvbox",
		Description: "Gruvbox: warm dark bg, orange accents, green strings",
		UserPrefix:  lipgloss.Color("#fe8019"),
		UserText:    lipgloss.Color("#ebdbb2"),
		Assistant:   lipgloss.Color("#b8bb26"),
		Thinking:    lipgloss.Color("#928374"),
		Tool:        lipgloss.Color("#fabd2f"),
		Error:       lipgloss.Color("#fb4934"),
		Status:      lipgloss.Color("#83a598"),
		Border:      lipgloss.Color("#504945"),
	},
	"nord": {
		Name:        "nord",
		Description: "Nord: cool blue palette with frost accents",
		UserPrefix:  lipgloss.Color("#88c0d0"),
		UserText:    lipgloss.Color("#eceff4"),
		Assistant:   lipgloss.Color("#d8dee9"),
		Thinking:    lipgloss.Color("#4c566a"),
		Tool:        lipgloss.Color("#8fbcbb"),
		Error:       lipgloss.Color("#bf616a"),
		Status:      lipgloss.Color("#81a1c1"),
		Border:      lipgloss.Color("#3b4252"),
	},
	"tokyonight": {
		Name:        "tokyonight",
		Description: "Tokyo Night: deep navy bg, purple accents, teal strings",
		UserPrefix:  lipgloss.Color("#bb9af7"),
		UserText:    lipgloss.Color("#c0caf5"),
		Assistant:   lipgloss.Color("#a9b1d6"),
		Thinking:    lipgloss.Color("#565f89"),
		Tool:        lipgloss.Color("#7dcfff"),
		Error:       lipgloss.Color("#f7768e"),
		Status:      lipgloss.Color("#7aa2f7"),
		Border:      lipgloss.Color("#1a1b26"),
	},
}

// ActiveTheme is the currently applied theme, swapped atomically so reads
// from concurrent renders are race-free.
var ActiveTheme atomic.Pointer[Theme]

func init() {
	t := Themes["dark"]
	ActiveTheme.Store(&t)
	applyToStyles(t)
}

// ApplyTheme switches the active palette and mutates the package-level style
// vars to use it. Returns an error if name is unknown.
func ApplyTheme(name string) error {
	t, ok := Themes[name]
	if !ok {
		return fmt.Errorf("unknown theme %q", name)
	}
	ActiveTheme.Store(&t)
	applyToStyles(t)
	return nil
}

// ListThemes returns themes sorted by name for display in pickers and /theme.
func ListThemes() []Theme {
	out := make([]Theme, 0, len(Themes))
	for _, t := range Themes {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// applyToStyles reassigns the package-level lipgloss styles using the colors
// from t. Lipgloss styles are immutable so each .Foreground/.Background call
// returns a fresh value that we store back.
func applyToStyles(t Theme) {
	userPrefixStyle = userPrefixStyle.Foreground(t.UserPrefix)
	userTextStyle = userTextStyle.Foreground(t.UserText)
	assistantStyle = assistantStyle.Foreground(t.Assistant)
	thinkingStyle = thinkingStyle.Foreground(t.Thinking)
	toolStartStyle = toolStartStyle.Foreground(t.Thinking)
	toolNameStyle = toolNameStyle.Foreground(t.Tool)
	toolResultStyle = toolResultStyle.Foreground(t.Thinking)
	errorStyle = errorStyle.Foreground(t.Error)

	statusModelStyle = statusModelStyle.Foreground(t.Status)
	statusUsageStyle = statusUsageStyle.Foreground(t.Tool)
	statusHintStyle = statusHintStyle.Foreground(t.Thinking)
	statusErrStyle = statusErrStyle.Foreground(t.Error)

	separatorStyle = separatorStyle.Foreground(t.Border)
	inputPromptStyle = inputPromptStyle.Foreground(t.UserPrefix)
	modalBorderStyle = modalBorderStyle.BorderForeground(t.Border)
	modalTitleStyle = modalTitleStyle.Foreground(t.Status)
	typeaheadBoxStyle = typeaheadBoxStyle.BorderForeground(t.Border)
	typeaheadItemStyle = typeaheadItemStyle.Foreground(t.Assistant)
	planBannerStyle = planBannerStyle.Foreground(t.Error)
}
