package ui

import (
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
)

// VimEnabled toggles whether the vim handler intercepts keys at all.
// VimMode is the current sub-mode: 0 = insert, 1 = normal.
//
// Both are exported so the /vim slash command (in another package) can
// flip the state without going through the Bubble Tea message loop.
var (
	VimEnabled atomic.Bool
	VimMode    atomic.Int32
)

// Vim sub-mode constants. Stored in VimMode as int32.
const (
	VimModeInsert int32 = 0
	VimModeNormal int32 = 1
)

// vimResult tells handleKey whether the vim handler consumed the key
// and, if so, what command (if any) to return back to Bubble Tea.
type vimResult struct {
	handled bool
	cmd     tea.Cmd
}

// pending operator state (e.g. the first 'd' in 'dd'). Reset whenever a
// non-matching key arrives. This is per-process state which is fine
// because there's only one Model in flight.
var vimPendingOp rune

// ToggleVim flips VimEnabled and returns the new value. When turning
// on, the mode is reset to insert so existing typing keeps working.
func ToggleVim() bool {
	now := !VimEnabled.Load()
	VimEnabled.Store(now)
	if now {
		VimMode.Store(VimModeInsert)
	} else {
		vimPendingOp = 0
	}
	return now
}

// handleVimKey is consulted at the top of Model.handleKey before the
// default Esc/Enter/etc. handling. It returns handled=false to let the
// default path run.
func handleVimKey(m *Model, key tea.KeyMsg) vimResult {
	if !VimEnabled.Load() {
		return vimResult{handled: false}
	}

	mode := VimMode.Load()

	if mode == VimModeInsert {
		// In insert mode the only key we care about is Esc, which
		// flips us to normal mode without clearing the input (the
		// default Esc handler would clear it).
		if key.Type == tea.KeyEsc {
			VimMode.Store(VimModeNormal)
			vimPendingOp = 0
			return vimResult{handled: true}
		}
		return vimResult{handled: false}
	}

	// Normal mode. PgUp/PgDn still scroll the viewport — we let those
	// fall through to the default handler.
	switch key.Type {
	case tea.KeyPgUp, tea.KeyPgDown:
		return vimResult{handled: false}
	}

	// Single-char commands. Bubble Tea encodes printable input as
	// KeyRunes with a Runes slice.
	if key.Type != tea.KeyRunes || len(key.Runes) == 0 {
		// Swallow other keys in normal mode so they don't leak
		// through to the textinput as literal characters.
		return vimResult{handled: true}
	}

	r := key.Runes[0]

	// Handle pending operator (currently only 'dd').
	if vimPendingOp == 'd' {
		vimPendingOp = 0
		if r == 'd' {
			m.input.SetValue("")
			m.refreshTypeahead()
		}
		return vimResult{handled: true}
	}

	pos := m.input.Position()
	val := m.input.Value()

	switch r {
	case 'h':
		if pos > 0 {
			m.input.SetCursor(pos - 1)
		}
	case 'l':
		if pos < len(val) {
			m.input.SetCursor(pos + 1)
		}
	case 'j', 'k':
		// Single-line input: j/k are no-ops on cursor, but we
		// still consume them so they don't insert literal chars.
	case 'i':
		VimMode.Store(VimModeInsert)
	case 'a':
		if pos < len(val) {
			m.input.SetCursor(pos + 1)
		}
		VimMode.Store(VimModeInsert)
	case '0':
		m.input.CursorStart()
	case '$':
		m.input.CursorEnd()
	case 'w':
		m.input.SetCursor(vimWordForward(val, pos))
	case 'b':
		m.input.SetCursor(vimWordBackward(val, pos))
	case 'x':
		if pos < len(val) {
			m.input.SetValue(val[:pos] + val[pos+1:])
			// Keep cursor at same logical column, clamping if at end.
			newPos := pos
			if newPos > len(m.input.Value()) {
				newPos = len(m.input.Value())
			}
			m.input.SetCursor(newPos)
			m.refreshTypeahead()
		}
	case 'd':
		// First half of 'dd' — wait for the next key.
		vimPendingOp = 'd'
	default:
		// Unrecognized key in normal mode — swallow silently.
	}

	return vimResult{handled: true}
}

// vimWordForward moves to the start of the next word boundary. A word
// boundary is the transition from a non-word character to a word
// character, where "word" means alnum or underscore (a simple
// heuristic, not a full vim emulation).
func vimWordForward(s string, pos int) int {
	n := len(s)
	if pos >= n {
		return n
	}
	i := pos
	// Skip current word.
	for i < n && isVimWordChar(s[i]) {
		i++
	}
	// Skip following non-word chars.
	for i < n && !isVimWordChar(s[i]) {
		i++
	}
	return i
}

// vimWordBackward moves to the start of the current or previous word.
func vimWordBackward(s string, pos int) int {
	if pos <= 0 {
		return 0
	}
	i := pos - 1
	// Skip preceding non-word chars.
	for i > 0 && !isVimWordChar(s[i]) {
		i--
	}
	// Walk to the start of this word.
	for i > 0 && isVimWordChar(s[i-1]) {
		i--
	}
	return i
}

func isVimWordChar(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z':
		return true
	case b >= 'A' && b <= 'Z':
		return true
	case b >= '0' && b <= '9':
		return true
	case b == '_':
		return true
	}
	return false
}
