package ui

import (
	"os"
	"strings"

	"claudecode/internal/chat"
	"claudecode/internal/core"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Bubble Tea messages produced by the event pump.
type uiEventMsg struct {
	ev   core.UIEvent
	more bool
	ch   <-chan core.UIEvent
}

type turnStartedMsg struct {
	ch <-chan core.UIEvent
}

type turnEndedMsg struct{}

type notifyMsg struct {
	ev core.UIEvent
	ch <-chan core.UIEvent
}

func pumpNotify(ch <-chan core.UIEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return notifyMsg{ev: ev, ch: ch}
	}
}

// Model is the root Bubble Tea model.
type Model struct {
	driver core.Driver

	viewport viewport.Model
	input    textinput.Model
	spinner  spinner.Model

	width  int
	height int

	// Streaming buffers for the in-flight assistant turn.
	liveText     strings.Builder
	liveThinking strings.Builder
	// Pending tool calls keyed by id, displayed inline while running.
	pendingTools map[string]toolPending

	inFlight bool
	usage    core.Usage
	errMsg   string
	logs     []string

	ready bool

	// Overlays. The topmost modal owns input until it dismisses.
	modals    []Modal
	typeahead *Typeahead
	cursor    int // selected typeahead row (mirrors typeahead.Selected())

	// Multiline input buffer (Alt+Enter commits a line into mlBuf).
	mlBuf multilineBuf

	// Transient toast notifications shown in the top-right corner.
	toasts []Toast

	// inputFocused tracks which pane gets text input. Currently always true
	// (input keeps focus); flipping it lets future work route keys to the
	// viewport for scroll-by-line gestures. Mouse clicks update it for
	// telemetry today.
	inputFocused bool
}

const maxLogs = 30

type toolPending struct {
	name  string
	input string
}

// NewModel constructs a Model and wires up its sub-components.
func NewModel(driver core.Driver) *Model {
	ti := textinput.New()
	ti.Placeholder = "type a message, or /help"
	ti.Prompt = ""
	ti.CharLimit = 0
	ti.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	vp := viewport.New(0, 0)
	vp.SetContent("")

	m := &Model{
		driver:       driver,
		viewport:     vp,
		input:        ti,
		spinner:      sp,
		pendingTools: make(map[string]toolPending),
		inputFocused: true,
	}
	m.typeahead = NewTypeahead(driver.Commands())
	return m
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
		pumpNotify(m.driver.Notifications()),
		scheduleToastTick(),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Modal stack owns input. Route the message to the top modal; if the
	// modal returns nil it has dismissed itself and we pop. Non-key
	// messages still pass through to the rest of the model so background
	// streams keep flowing while a modal is up.
	if len(m.modals) > 0 {
		if _, ok := msg.(tea.KeyMsg); ok {
			top := m.modals[len(m.modals)-1]
			next, cmd := top.Update(msg)
			if next == nil {
				m.modals = m.modals[:len(m.modals)-1]
			} else {
				m.modals[len(m.modals)-1] = next
			}
			return m, cmd
		}
	}

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		m.refreshViewport()
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case toastTickMsg:
		// Periodic prune; reschedule unconditionally so the loop keeps ticking.
		m.pruneToasts()
		cmds = append(cmds, scheduleToastTick())

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.inFlight {
			cmds = append(cmds, cmd)
		}

	case turnStartedMsg:
		m.inFlight = true
		m.errMsg = ""
		m.liveText.Reset()
		m.liveThinking.Reset()
		m.pendingTools = make(map[string]toolPending)
		cmds = append(cmds, pumpEvents(msg.ch), m.spinner.Tick)
		m.refreshViewport()

	case uiEventMsg:
		m.applyEvent(msg.ev)
		m.refreshViewport()
		if msg.more {
			cmds = append(cmds, pumpEvents(msg.ch))
		} else {
			m.inFlight = false
			// Snapshot now contains any final assistant message; clear live buffers.
			m.liveText.Reset()
			m.liveThinking.Reset()
			m.pendingTools = make(map[string]toolPending)
			m.refreshViewport()
		}

	case turnEndedMsg:
		m.inFlight = false
		m.refreshViewport()

	case notifyMsg:
		m.applyEvent(msg.ev)
		m.refreshViewport()
		cmds = append(cmds, pumpNotify(msg.ch))
	}

	// Forward to viewport for built-in scrolling support of mouse/wheel etc.
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	if vpCmd != nil {
		cmds = append(cmds, vpCmd)
	}

	// Forward to input.
	var inCmd tea.Cmd
	m.input, inCmd = m.input.Update(msg)
	if inCmd != nil {
		cmds = append(cmds, inCmd)
	}

	// Recompute typeahead suggestions after every input change.
	m.refreshTypeahead()

	return m, tea.Batch(cmds...)
}

// refreshTypeahead syncs the typeahead helper with the current input value.
func (m *Model) refreshTypeahead() {
	if m.typeahead == nil {
		return
	}
	m.typeahead.Update(m.input.Value())
	m.cursor = m.typeahead.Selected()
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		if m.inFlight {
			m.driver.Cancel()
			return m, nil
		}
		return m, tea.Quit

	case tea.KeyCtrlF:
		// Open the search overlay over the current snapshot.
		m.modals = append(m.modals, NewSearchOverlay(m))
		return m, nil
	}

	// Alt+Enter (and Shift+Enter on terminals that report it as Alt) commits
	// the current line into the multiline buffer instead of submitting.
	if isMultilineNewlineKey(msg) {
		m.mlBuf.commit(m.input.Value())
		m.input.SetValue("")
		m.layout()
		m.refreshTypeahead()
		return m, nil
	}

	// Ctrl+, (when the terminal forwards it) opens the settings editor.
	if msg.String() == "ctrl+," {
		m.modals = append(m.modals, NewSettingsModal())
		return m, nil
	}

	// Vim mode (when enabled) intercepts before the default Esc/printable
	// handling so normal-mode commands and the insert-mode Esc remap take
	// effect. PgUp/PgDn are explicitly let through by handleVimKey.
	if r := handleVimKey(m, msg); r.handled {
		return m, r.cmd
	}

	switch msg.Type {
	case tea.KeyEsc:
		m.input.SetValue("")
		m.refreshTypeahead()
		return m, nil

	case tea.KeyPgUp:
		m.viewport.HalfViewUp()
		return m, nil

	case tea.KeyPgDown:
		m.viewport.HalfViewDown()
		return m, nil

	case tea.KeyCtrlAt:
		// Ctrl+@ explicitly opens the file picker.
		return m, m.openFilePicker()

	case tea.KeyTab:
		// Tab cycles through typeahead suggestions when the input is a
		// slash command. We do NOT auto-complete the input: cycling lets
		// the user inspect synopses; pressing Enter submits whatever is
		// currently typed.
		if m.typeahead != nil && len(m.typeahead.Suggestions()) > 0 &&
			strings.HasPrefix(m.input.Value(), "/") {
			m.typeahead.Cycle(1)
			m.cursor = m.typeahead.Selected()
			return m, nil
		}
		// If the current input ends with @<word> with no whitespace, Tab
		// opens the file picker so the user can select a file by name.
		if last := lastAtToken(m.input.Value()); last != "" || endsWithBareAt(m.input.Value()) {
			return m, m.openFilePicker()
		}

	case tea.KeyShiftTab:
		if m.typeahead != nil && len(m.typeahead.Suggestions()) > 0 &&
			strings.HasPrefix(m.input.Value(), "/") {
			m.typeahead.Cycle(-1)
			m.cursor = m.typeahead.Selected()
			return m, nil
		}

	case tea.KeyEnter:
		return m.submit()
	}

	// Default: let viewport/input handle in main Update loop. We re-route by
	// returning here without consuming so the regular forwarding fires next
	// time. To keep one path, just delegate input here.
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.refreshTypeahead()
	return m, cmd
}

func (m *Model) submit() (tea.Model, tea.Cmd) {
	full := m.mlBuf.fullText(m.input.Value())
	text := strings.TrimRight(full, " \t\n")
	if text == "" {
		return m, nil
	}
	m.input.SetValue("")
	m.mlBuf.reset()
	m.layout()

	if strings.HasPrefix(text, "/") {
		// Special case: /help opens a TextModal of all known commands rather
		// than dispatching as a regular command (gives a scrollable view).
		if strings.TrimSpace(text) == "/help" {
			body := renderHelpText(m.driver.Commands())
			m.modals = append(m.modals, &TextModal{
				TitleText: "Help",
				Body:      body,
			})
			return m, nil
		}
		line := strings.TrimPrefix(text, "/")
		if err := m.driver.RunCommand(line); err != nil {
			m.errMsg = err.Error()
		} else {
			m.errMsg = ""
		}
		m.refreshViewport()
		return m, nil
	}

	if m.inFlight {
		// Avoid stacking concurrent turns; ignore until current is done.
		m.errMsg = "turn in progress; press Ctrl+C to cancel"
		return m, nil
	}

	ch := m.driver.Submit(text)
	return m, func() tea.Msg { return turnStartedMsg{ch: ch} }
}

func (m *Model) applyEvent(ev core.UIEvent) {
	switch e := ev.(type) {
	case core.UIAssistantTextDeltaEvent:
		m.liveText.WriteString(e.Text)

	case core.UIThinkingDeltaEvent:
		m.liveThinking.WriteString(e.Text)

	case core.UIToolStartEvent:
		m.pendingTools[e.ID] = toolPending{name: e.Name, input: e.Input}

	case core.UIToolResultEvent:
		// Drop from pending; the snapshot will hold the canonical record.
		delete(m.pendingTools, e.ID)

	case core.UITurnDoneEvent:
		m.usage = e.Usage

	case core.UIStatusEvent:
		if e.Level == core.NotifyError {
			m.errMsg = e.Text
			m.appendLog("! " + e.Text)
		} else {
			m.appendLog(e.Text)
		}
		m.addToast(e.Level, e.Text)

	case core.UIErrorEvent:
		if e.Err != nil {
			m.errMsg = e.Err.Error()
			m.appendLog("! " + e.Err.Error())
			m.addToast(core.NotifyError, e.Err.Error())
		}

	case core.UIPermissionPromptEvent:
		// Push a modal onto the stack so the user can decide. The Reply
		// channel is closed by the chat side once it consumes a value.
		m.modals = append(m.modals, &PermissionModal{
			Tool:      e.Tool,
			InputJSON: e.InputJSON,
			Reply:     e.ReplyChan,
		})

	case core.UIAskUserEvent:
		// Tool requested a free-form user response. Push a single-line
		// prompt; Esc cancels by sending "" to the reply channel.
		m.modals = append(m.modals, NewAskUserModal(e.Question, e.Reply))
	}
}

// pumpEvents reads one event from the channel and re-issues itself for the
// next, signalling end-of-turn when the channel closes.
func pumpEvents(ch <-chan core.UIEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return uiEventMsg{ev: nil, more: false, ch: ch}
		}
		return uiEventMsg{ev: ev, more: true, ch: ch}
	}
}

// handleMouse routes mouse events. Wheel events scroll the viewport by half
// a page; clicks update the focus marker so the model knows whether the
// next typed key is meant for the input or (in future) the viewport.
func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.viewport.HalfViewUp()
		return m, nil
	case tea.MouseButtonWheelDown:
		m.viewport.HalfViewDown()
		return m, nil
	}

	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		// Reserve the bottom three rows (separator + status + input) for the
		// input pane; everything above is viewport. The focus flag is logged
		// today; future work can route key events accordingly.
		inputTop := m.height - 3
		if msg.Y >= inputTop {
			m.inputFocused = true
			m.input.Focus()
			m.appendLog("focus: input")
		} else {
			m.inputFocused = false
			m.appendLog("focus: viewport")
		}
		m.refreshViewport()
		return m, nil
	}

	return m, nil
}

func (m *Model) layout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	// Reserve: 1 line separator, 1 line status bar, 1 line input + extra
	// rows for any committed multiline lines so the viewport shrinks rather
	// than overflowing.
	extra := len(m.mlBuf.lines)
	vpHeight := m.height - 3 - extra
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.viewport.Width = m.width
	m.viewport.Height = vpHeight
	m.input.Width = m.width - 2 // leave room for "> "
}

func (m *Model) refreshViewport() {
	hist := m.driver.Snapshot()
	body := renderHistory(hist, m.viewport.Width)

	// Append in-flight content at the bottom.
	var extras []string
	for _, t := range m.pendingTools {
		extras = append(extras, renderToolStart(t.name, t.input))
	}
	if m.liveThinking.Len() > 0 {
		extras = append(extras, thinkingStyle.Render("thinking: "+softWrap(m.liveThinking.String(), m.viewport.Width)))
	}
	if m.liveText.Len() > 0 {
		extras = append(extras, assistantStyle.Render(softWrap(m.liveText.String(), m.viewport.Width)))
	}
	if len(m.logs) > 0 {
		extras = append(extras, renderLogs(m.logs, m.viewport.Width))
	}
	if len(extras) > 0 {
		if body != "" {
			body += "\n"
		}
		body += strings.Join(extras, "\n")
	}

	m.viewport.SetContent(body)
	m.viewport.GotoBottom()
}

func (m *Model) appendLog(s string) {
	m.logs = append(m.logs, s)
	if len(m.logs) > maxLogs {
		m.logs = m.logs[len(m.logs)-maxLogs:]
	}
}

func (m *Model) View() string {
	if !m.ready {
		return "initializing..."
	}

	model := "unknown"
	if sess := m.driver.Session(); sess != nil {
		model = sess.Model()
	}

	prompt := "> "
	if m.inFlight {
		prompt = m.spinner.View() + " "
	}
	if VimEnabled.Load() {
		tag := "[I] "
		if VimMode.Load() == VimModeNormal {
			tag = "[N] "
		}
		prompt = tag + prompt
	}

	status := renderStatusBar(m.width, model, m.usage, m.errMsg)
	input := renderMultilineInput(&m.mlBuf, prompt, m.input.View(), m.width)
	sep := renderSeparator(m.width)

	parts := make([]string, 0, 6)
	if chat.PlanModeActive.Load() {
		parts = append(parts, planBannerStyle.Render("═══ PLAN MODE — mutating tools blocked ═══"))
	}
	parts = append(parts, m.viewport.View(), sep)
	if ta := m.typeaheadView(); ta != "" {
		parts = append(parts, ta)
	}
	parts = append(parts, status, input)

	base := strings.Join(parts, "\n")

	// Overlay toasts (top-right corner) on top of the base view.
	if active := m.activeToasts(); len(active) > 0 {
		base = overlayToasts(base, active, m.width)
	}

	// Overlay the topmost modal, if any. We append below the base view
	// rather than compositing — Bubble Tea doesn't expose absolute
	// positioning so this keeps things simple and predictable.
	if len(m.modals) > 0 {
		top := m.modals[len(m.modals)-1]
		modalView := top.View(m.width, m.height)
		base = base + "\n" + modalView
	}

	return base
}

// openFilePicker opens a FilePickerModal at the process cwd and, on select,
// rewrites the trailing @<word> token of the input with the chosen path. If
// the input has no @<word> token yet, the path is appended preceded by `@`.
func (m *Model) openFilePicker() tea.Cmd {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	picker := NewFilePickerModal(cwd, func(path string) tea.Cmd {
		return m.onFilePicked(path)
	})
	m.modals = append(m.modals, picker)
	return nil
}

// onFilePicked replaces the trailing @<token> with the chosen absolute path
// or appends `@<path>` if no token is present.
func (m *Model) onFilePicked(path string) tea.Cmd {
	val := m.input.Value()
	if endsWithBareAt(val) {
		m.input.SetValue(val + path)
	} else if tok := lastAtToken(val); tok != "" {
		// Strip the trailing @<tok>... and append @<path>.
		idx := strings.LastIndex(val, "@")
		if idx >= 0 {
			m.input.SetValue(val[:idx] + "@" + path)
		} else {
			m.input.SetValue(val + " @" + path)
		}
	} else {
		if val != "" && !strings.HasSuffix(val, " ") {
			val += " "
		}
		m.input.SetValue(val + "@" + path)
	}
	m.input.CursorEnd()
	return nil
}

// lastAtToken returns the text after the final `@` if and only if that final
// `@` is followed by at least one non-space character and no whitespace. An
// empty string indicates no eligible token.
func lastAtToken(s string) string {
	idx := strings.LastIndex(s, "@")
	if idx < 0 {
		return ""
	}
	tail := s[idx+1:]
	if tail == "" {
		return ""
	}
	if strings.ContainsAny(tail, " \t\n") {
		return ""
	}
	return tail
}

// endsWithBareAt reports whether the input ends with a literal `@` with no
// following characters (i.e. the user just typed the trigger).
func endsWithBareAt(s string) bool {
	return strings.HasSuffix(s, "@")
}

// typeaheadView returns the rendered typeahead panel or "" if not active.
func (m *Model) typeaheadView() string {
	if m.typeahead == nil {
		return ""
	}
	if !strings.HasPrefix(m.input.Value(), "/") {
		return ""
	}
	return m.typeahead.View(m.width)
}
