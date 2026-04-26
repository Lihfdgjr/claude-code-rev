package ui

import "github.com/charmbracelet/lipgloss"

var (
	userPrefixStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Bold(true)

	userTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	thinkingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true)

	toolStartStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	toolNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("111")).
			Bold(true)

	toolResultStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("242")).
			PaddingLeft(2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("231")).
			Background(lipgloss.Color("237")).
			Padding(0, 1)

	statusModelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("117")).
				Background(lipgloss.Color("237")).
				Bold(true)

	statusUsageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("186")).
				Background(lipgloss.Color("237"))

	statusHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Background(lipgloss.Color("237"))

	statusErrStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Background(lipgloss.Color("237")).
			Bold(true)

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	inputPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("14")).
				Bold(true)

	modalBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("117")).
				Padding(0, 1)

	modalTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117")).
			Bold(true).
			MarginBottom(1)

	typeaheadBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)

	typeaheadItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	typeaheadSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("231")).
				Background(lipgloss.Color("24")).
				Bold(true)

	planBannerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	toastInfoStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("117")).
			Foreground(lipgloss.Color("231")).
			Padding(0, 1).
			MaxWidth(toastMaxWidth)

	toastWarnStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("214")).
			Foreground(lipgloss.Color("231")).
			Padding(0, 1).
			MaxWidth(toastMaxWidth)

	toastErrStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("9")).
			Foreground(lipgloss.Color("231")).
			Bold(true).
			Padding(0, 1).
			MaxWidth(toastMaxWidth)

	toastDebugStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("244")).
			Foreground(lipgloss.Color("249")).
			Padding(0, 1).
			MaxWidth(toastMaxWidth)

	statusIconOKStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10")).
				Background(lipgloss.Color("237")).
				Bold(true)

	statusIconErrStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("9")).
				Background(lipgloss.Color("237")).
				Bold(true)

	statusModeVimStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("213")).
				Background(lipgloss.Color("237")).
				Bold(true)

	statusModePlanStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("9")).
				Background(lipgloss.Color("237")).
				Bold(true)

	markdownH1Style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("213")).
			Bold(true)

	markdownH2Style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117")).
			Bold(true)

	markdownH3Style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true)

	markdownBoldStyle = lipgloss.NewStyle().
				Bold(true)

	markdownItalicStyle = lipgloss.NewStyle().
				Italic(true)

	markdownCodeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("14")).
				Background(lipgloss.Color("236"))

	markdownLinkStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("117")).
				Underline(true)

	markdownQuoteStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Italic(true)

	markdownListPrefixStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	citationStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true)
)

// lipglossWidth is a tiny wrapper over lipgloss.Width so callers can use it
// without needing a direct lipgloss import where they only care about width.
func lipglossWidth(s string) int { return lipgloss.Width(s) }
