package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// SettingKind classifies the editing UI for a SettingItem. The "enum:" prefix
// is followed by a comma-separated list of allowed values.
type SettingKind string

const (
	SettingKindString   SettingKind = "string"
	SettingKindBool     SettingKind = "bool"
	SettingKindInt      SettingKind = "int"
	SettingKindReadOnly SettingKind = "readonly"
)

// SettingItem describes one editable row of the SettingsModal.
type SettingItem struct {
	Label string
	Path  []string
	Value string
	Kind  SettingKind
	// Mask hides the displayed value (e.g. for api_key).
	Mask bool
}

// SettingsModal is a vertical list of SettingItems with inline editing. The
// modified values are written back to ~/.claude/settings.json on save.
type SettingsModal struct {
	Cursor   int
	Items    []SettingItem
	FilePath string
	Dirty    bool

	editing bool
	input   textinput.Model
	confirm bool
	status  string
}

// NewSettingsModal constructs the modal, reading current values from
// ~/.claude/settings.json on a best-effort basis.
func NewSettingsModal() *SettingsModal {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".claude", "settings.json")
	current := readSettingsFile(path)

	themeNames := make([]string, 0, len(ListThemes()))
	for _, t := range ListThemes() {
		themeNames = append(themeNames, t.Name)
	}
	themeEnum := SettingKind("enum:" + strings.Join(themeNames, ","))

	items := []SettingItem{
		{Label: "API Key", Path: []string{"api_key"}, Kind: SettingKindString, Mask: true},
		{Label: "Base URL", Path: []string{"base_url"}, Kind: SettingKindString},
		{Label: "Model", Path: []string{"model"},
			Kind: SettingKind("enum:claude-opus-4-7,claude-sonnet-4-6,claude-haiku-4-5-20251001,claude-opus-4-5")},
		{Label: "Permissions Mode", Path: []string{"permissions", "mode"},
			Kind: SettingKind("enum:allow,deny,ask")},
		{Label: "Theme", Path: []string{"theme"}, Kind: themeEnum},
		{Label: "Thinking", Path: []string{"thinking", "enabled"}, Kind: SettingKindBool},
		{Label: "Vim Mode", Path: []string{"vim", "enabled"}, Kind: SettingKindBool},
		{Label: "SessionStart Hook", Path: []string{"hooks", "session_start"}, Kind: SettingKindString},
		{Label: "MCP Servers", Path: []string{"mcp", "servers"}, Kind: SettingKindReadOnly},
		{Label: "Permissions Allow", Path: []string{"permissions", "allow"}, Kind: SettingKindReadOnly},
		{Label: "Permissions Deny", Path: []string{"permissions", "deny"}, Kind: SettingKindReadOnly},
		{Label: "Telemetry Path", Path: []string{"telemetry", "path"}, Kind: SettingKindString},
	}
	for i := range items {
		if items[i].Kind == SettingKindReadOnly {
			items[i].Value = readOnlyDisplay(current, items[i].Path)
		} else {
			items[i].Value = lookupValue(current, items[i].Path)
		}
	}

	ti := textinput.New()
	ti.Prompt = "= "
	ti.CharLimit = 0

	return &SettingsModal{
		Items:    items,
		FilePath: path,
		input:    ti,
	}
}

func (m *SettingsModal) Init() tea.Cmd { return textinput.Blink }

func (m *SettingsModal) Title() string { return "Settings" }

func (m *SettingsModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.confirm {
		// Dirty-discard confirmation: y throws away changes, anything else
		// returns to the editor.
		if key.Type == tea.KeyRunes && len(key.Runes) > 0 &&
			strings.ToLower(string(key.Runes)) == "y" {
			return nil, nil
		}
		m.confirm = false
		return m, nil
	}

	if m.editing {
		return m.updateEditing(key)
	}

	switch key.Type {
	case tea.KeyEsc:
		if m.Dirty {
			m.confirm = true
			return m, nil
		}
		return nil, nil
	case tea.KeyUp:
		if m.Cursor > 0 {
			m.Cursor--
		}
		return m, nil
	case tea.KeyDown:
		if m.Cursor < len(m.Items)-1 {
			m.Cursor++
		}
		return m, nil
	case tea.KeyEnter:
		m.beginEdit()
		return m, textinput.Blink
	case tea.KeyRunes:
		if len(key.Runes) == 0 {
			return m, nil
		}
		switch strings.ToLower(string(key.Runes)) {
		case "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "j":
			if m.Cursor < len(m.Items)-1 {
				m.Cursor++
			}
		case "s":
			if err := m.save(); err != nil {
				m.status = "save failed: " + err.Error()
			} else {
				m.Dirty = false
				m.status = "saved to " + m.FilePath
			}
		case "q":
			if m.Dirty {
				m.confirm = true
				return m, nil
			}
			return nil, nil
		}
	}
	return m, nil
}

func (m *SettingsModal) beginEdit() {
	if m.Cursor < 0 || m.Cursor >= len(m.Items) {
		return
	}
	it := m.Items[m.Cursor]
	if it.Kind == SettingKindReadOnly {
		return
	}
	if it.Kind == SettingKindBool {
		// Enter toggles bool values in place.
		next := "true"
		if normalizeBool(it.Value) == "true" {
			next = "false"
		}
		m.Items[m.Cursor].Value = next
		m.Dirty = true
		m.status = ""
		return
	}
	if values := enumValues(it.Kind); len(values) > 0 {
		// Cycle to the next value rather than typing.
		next := nextEnum(values, it.Value)
		if next != it.Value {
			m.Items[m.Cursor].Value = next
			m.Dirty = true
			m.status = ""
		}
		return
	}
	m.editing = true
	m.input.SetValue(it.Value)
	m.input.Focus()
	m.input.CursorEnd()
}

func (m *SettingsModal) updateEditing(key tea.KeyMsg) (Modal, tea.Cmd) {
	switch key.Type {
	case tea.KeyEnter:
		v := m.input.Value()
		if m.Items[m.Cursor].Kind == SettingKindBool {
			v = normalizeBool(v)
		}
		if m.Items[m.Cursor].Value != v {
			m.Items[m.Cursor].Value = v
			m.Dirty = true
		}
		m.editing = false
		m.input.Blur()
		m.status = ""
		return m, nil
	case tea.KeyEsc:
		m.editing = false
		m.input.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(key)
	return m, cmd
}

func (m *SettingsModal) save() error {
	current := readSettingsFile(m.FilePath)
	if current == nil {
		current = map[string]any{}
	}
	for _, it := range m.Items {
		if it.Kind == SettingKindReadOnly {
			continue
		}
		if it.Value == "" {
			deleteValue(current, it.Path)
			continue
		}
		setValue(current, it.Path, parseValueForKind(it.Value, it.Kind))
	}
	if err := os.MkdirAll(filepath.Dir(m.FilePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(m.FilePath, data, 0o600); err != nil {
		return err
	}
	// Apply the theme immediately if set so the change is visible.
	for _, it := range m.Items {
		if len(it.Path) == 1 && it.Path[0] == "theme" && it.Value != "" {
			_ = ApplyTheme(it.Value)
		}
	}
	return nil
}

func (m *SettingsModal) View(width, height int) string {
	w := width
	if w > 80 {
		w = 80
	}

	if m.confirm {
		body := "You have unsaved changes. Discard and close? [y/N]"
		return centerModal(renderModalFrame(m.Title(), body, w), width, height)
	}

	var b strings.Builder
	b.WriteString("File: " + m.FilePath + "\n\n")
	for i, it := range m.Items {
		display := it.Value
		if it.Mask && display != "" {
			display = maskValue(display)
		}
		if display == "" {
			display = "(unset)"
		}
		row := fmt.Sprintf("%-20s %s", it.Label, display)
		if values := enumValues(it.Kind); len(values) > 0 {
			row += "  " + thinkingStyle.Render("["+strings.Join(values, "|")+"]")
		} else if it.Kind == SettingKindBool {
			row += "  " + thinkingStyle.Render("[true|false]")
		} else if it.Kind == SettingKindReadOnly {
			row += "  " + thinkingStyle.Render("(read-only)")
		}
		if i == m.Cursor {
			b.WriteString(typeaheadSelectedStyle.Render("> " + row))
		} else {
			b.WriteString(typeaheadItemStyle.Render("  " + row))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.editing {
		b.WriteString(m.input.View())
		b.WriteString("\n")
		b.WriteString(thinkingStyle.Render("(Enter accept, Esc cancel)"))
	} else {
		hint := "(Up/Down move, Enter edit/cycle, s save, Esc close)"
		if m.Dirty {
			hint = "[modified] " + hint
		}
		b.WriteString(thinkingStyle.Render(hint))
	}
	if m.status != "" {
		b.WriteString("\n")
		b.WriteString(thinkingStyle.Render(m.status))
	}

	frame := renderModalFrame(m.Title(), strings.TrimRight(b.String(), "\n"), w)
	return centerModal(frame, width, height)
}

// --- helpers ---

func readSettingsFile(path string) map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]any{}
	}
	return m
}

func lookupValue(m map[string]any, path []string) string {
	if len(path) == 0 {
		return ""
	}
	cur := any(m)
	for _, p := range path {
		obj, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur, ok = obj[p]
		if !ok {
			return ""
		}
	}
	switch v := cur.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		return fmt.Sprintf("%g", v)
	case nil:
		return ""
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func setValue(m map[string]any, path []string, val any) {
	if len(path) == 0 {
		return
	}
	cur := m
	for i, p := range path {
		if i == len(path)-1 {
			cur[p] = val
			return
		}
		next, ok := cur[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[p] = next
		}
		cur = next
	}
}

func deleteValue(m map[string]any, path []string) {
	if len(path) == 0 {
		return
	}
	cur := m
	for i, p := range path {
		if i == len(path)-1 {
			delete(cur, p)
			return
		}
		next, ok := cur[p].(map[string]any)
		if !ok {
			return
		}
		cur = next
	}
}

func parseValueForKind(v string, kind SettingKind) any {
	switch kind {
	case SettingKindBool:
		return v == "true"
	case SettingKindInt:
		var n int
		_, err := fmt.Sscanf(v, "%d", &n)
		if err != nil {
			return v
		}
		return n
	default:
		return v
	}
}

func normalizeBool(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "y", "on":
		return "true"
	default:
		return "false"
	}
}

func enumValues(k SettingKind) []string {
	s := string(k)
	if !strings.HasPrefix(s, "enum:") {
		return nil
	}
	return strings.Split(strings.TrimPrefix(s, "enum:"), ",")
}

func nextEnum(values []string, current string) string {
	if len(values) == 0 {
		return current
	}
	for i, v := range values {
		if v == current {
			return values[(i+1)%len(values)]
		}
	}
	return values[0]
}

func maskValue(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return strings.Repeat("*", len(s)-4) + s[len(s)-4:]
}

// readOnlyDisplay renders a read-only field. For mcp.servers it shows the
// server count; for permissions allow/deny it joins the list with commas.
func readOnlyDisplay(m map[string]any, path []string) string {
	if len(path) == 0 {
		return ""
	}
	cur := any(m)
	for _, p := range path {
		obj, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur, ok = obj[p]
		if !ok {
			return ""
		}
	}
	switch v := cur.(type) {
	case map[string]any:
		return fmt.Sprintf("%d configured", len(v))
	case []any:
		parts := make([]string, 0, len(v))
		for _, x := range v {
			parts = append(parts, fmt.Sprint(x))
		}
		return strings.Join(parts, ",")
	case string:
		return v
	case nil:
		return ""
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}
