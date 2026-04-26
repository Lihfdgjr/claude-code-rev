package computeruse

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ErrNotSupported is returned when an operation is not implemented on the host OS.
var ErrNotSupported = errors.New("computeruse: not supported in this build")

// Computer is the desktop-automation surface exposed to tools.
type Computer interface {
	Screenshot(ctx context.Context) ([]byte, error)
	Click(ctx context.Context, x, y int, button string) error
	Type(ctx context.Context, text string) error
	Key(ctx context.Context, key string) error
	Move(ctx context.Context, x, y int) error
	Scroll(ctx context.Context, x, y, dx, dy int) error
}

type winComputer struct{}

// New returns a Computer backed by PowerShell on Windows.
func New() Computer { return winComputer{} }

const psTimeout = 10 * time.Second

func runPowerShell(ctx context.Context, script string) ([]byte, error) {
	if runtime.GOOS != "windows" {
		return nil, ErrNotSupported
	}
	cctx, cancel := context.WithTimeout(ctx, psTimeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", "-")
	cmd.Stdin = strings.NewReader(script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("powershell: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func notWindowsErr(op string) error {
	return fmt.Errorf("computeruse: %s only implemented on Windows in this build", op)
}

func (winComputer) Screenshot(ctx context.Context) ([]byte, error) {
	if runtime.GOOS != "windows" {
		return nil, notWindowsErr("Screenshot")
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("ccshot_%d.png", time.Now().UnixNano()))
	psPath := strings.ReplaceAll(tmp, `\`, `\\`)
	script := `
Add-Type -AssemblyName System.Windows.Forms,System.Drawing
$bounds = [System.Windows.Forms.SystemInformation]::VirtualScreen
$bmp = New-Object System.Drawing.Bitmap $bounds.Width, $bounds.Height
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.CopyFromScreen($bounds.Location, [System.Drawing.Point]::Empty, $bounds.Size)
$bmp.Save("` + psPath + `", [System.Drawing.Imaging.ImageFormat]::Png)
$g.Dispose()
$bmp.Dispose()
`
	if _, err := runPowerShell(ctx, script); err != nil {
		return nil, err
	}
	defer os.Remove(tmp)
	return os.ReadFile(tmp)
}

func (winComputer) Click(ctx context.Context, x, y int, button string) error {
	if runtime.GOOS != "windows" {
		return notWindowsErr("Click")
	}
	var down, up uint32
	switch strings.ToLower(button) {
	case "", "left":
		down, up = 0x0002, 0x0004
	case "right":
		down, up = 0x0008, 0x0010
	case "middle":
		return errors.New("computeruse: middle button not supported")
	default:
		return fmt.Errorf("computeruse: unknown button %q", button)
	}
	script := `
Add-Type -MemberDefinition '[DllImport("user32.dll",CallingConvention=CallingConvention.StdCall)]
public static extern void mouse_event(uint dwFlags, uint dx, uint dy, uint dwData, int dwExtraInfo);' -Name U32 -Namespace W
Add-Type -AssemblyName System.Windows.Forms,System.Drawing
[System.Windows.Forms.Cursor]::Position = New-Object System.Drawing.Point(` + strconv.Itoa(x) + `,` + strconv.Itoa(y) + `)
[W.U32]::mouse_event(` + strconv.FormatUint(uint64(down), 10) + `,0,0,0,0)
Start-Sleep -Milliseconds 30
[W.U32]::mouse_event(` + strconv.FormatUint(uint64(up), 10) + `,0,0,0,0)
`
	_, err := runPowerShell(ctx, script)
	return err
}

// escapeSendKeys wraps SendKeys metacharacters in braces.
func escapeSendKeys(text string) string {
	var b strings.Builder
	for _, r := range text {
		switch r {
		case '+', '^', '%', '~', '(', ')', '{', '}', '[', ']':
			b.WriteByte('{')
			b.WriteRune(r)
			b.WriteByte('}')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func psSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func (winComputer) Type(ctx context.Context, text string) error {
	if runtime.GOOS != "windows" {
		return notWindowsErr("Type")
	}
	script := `
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.SendKeys]::SendWait(` + psSingleQuote(escapeSendKeys(text)) + `)
`
	_, err := runPowerShell(ctx, script)
	return err
}

var keyAliases = map[string]string{
	"enter":     "{ENTER}",
	"return":    "{ENTER}",
	"escape":    "{ESC}",
	"esc":       "{ESC}",
	"tab":       "{TAB}",
	"backspace": "{BACKSPACE}",
	"bksp":      "{BACKSPACE}",
	"delete":    "{DELETE}",
	"del":       "{DELETE}",
	"home":      "{HOME}",
	"end":       "{END}",
	"pageup":    "{PGUP}",
	"pagedown":  "{PGDN}",
	"up":        "{UP}",
	"down":      "{DOWN}",
	"left":      "{LEFT}",
	"right":     "{RIGHT}",
	"space":     " ",
	"f1":        "{F1}", "f2": "{F2}", "f3": "{F3}", "f4": "{F4}",
	"f5": "{F5}", "f6": "{F6}", "f7": "{F7}", "f8": "{F8}",
	"f9": "{F9}", "f10": "{F10}", "f11": "{F11}", "f12": "{F12}",
}

func (winComputer) Key(ctx context.Context, key string) error {
	if runtime.GOOS != "windows" {
		return notWindowsErr("Key")
	}
	mapped, ok := keyAliases[strings.ToLower(strings.TrimSpace(key))]
	if !ok {
		mapped = escapeSendKeys(key)
	}
	script := `
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.SendKeys]::SendWait(` + psSingleQuote(mapped) + `)
`
	_, err := runPowerShell(ctx, script)
	return err
}

func (winComputer) Move(ctx context.Context, x, y int) error {
	if runtime.GOOS != "windows" {
		return notWindowsErr("Move")
	}
	script := `
Add-Type -AssemblyName System.Windows.Forms,System.Drawing
[System.Windows.Forms.Cursor]::Position = New-Object System.Drawing.Point(` + strconv.Itoa(x) + `,` + strconv.Itoa(y) + `)
`
	_, err := runPowerShell(ctx, script)
	return err
}

func (winComputer) Scroll(ctx context.Context, x, y, dx, dy int) error {
	if runtime.GOOS != "windows" {
		return notWindowsErr("Scroll")
	}
	// MOUSEEVENTF_WHEEL = 0x0800; positive dy = scroll up; one notch = 120.
	delta := dy * 120
	script := `
Add-Type -MemberDefinition '[DllImport("user32.dll",CallingConvention=CallingConvention.StdCall)]
public static extern void mouse_event(uint dwFlags, uint dx, uint dy, int dwData, int dwExtraInfo);' -Name U32S -Namespace W
Add-Type -AssemblyName System.Windows.Forms,System.Drawing
[System.Windows.Forms.Cursor]::Position = New-Object System.Drawing.Point(` + strconv.Itoa(x) + `,` + strconv.Itoa(y) + `)
[W.U32S]::mouse_event(0x0800,0,0,` + strconv.Itoa(delta) + `,0)
`
	_, err := runPowerShell(ctx, script)
	return err
}
