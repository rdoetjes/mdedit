//go:build !js

package ui

// The desktop side of everything that differs between the desktop and the
// browser (platform_js.go): files, settings, links and quitting. Both files
// define the same functions and types.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"gioui.org/font"
)

// platformState holds what one platform needs beyond the shared App
// fields; the desktop needs nothing.
type platformState struct{}

// platformFonts are fonts beyond the Go fonts; desktop systems have their
// own fonts for symbols the Go fonts lack.
func platformFonts() []font.FontFace { return nil }

// handleArrivals applies results that other goroutines delivered; on the
// desktop everything happens on the frame goroutine.
func (a *App) handleArrivals() {}

// autosave keeps the text between sessions; the desktop has files, and
// saveRecovery for unsaved work.
func (a *App) autosave() {}

// quitCommands are the File menu's last items.
func (a *App) quitCommands() []*command {
	return []*command{nil, newCommand("Quit", "Q", 0, a.cmdQuit)}
}

func (a *App) loadInitial(path string) error {
	if path == "" {
		a.load("", "")
		return nil
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		a.load(path, string(data))
	case os.IsNotExist(err):
		a.load("", "")
		a.path = path
		a.preview.BaseDir = filepath.Dir(path)
	default:
		return err
	}
	return nil
}

// saveRecovery keeps unsaved work when the window is closed with changes.
func (a *App) saveRecovery() {
	if !a.dirty {
		return
	}
	p, err := a.recoveryPath()
	if err == nil {
		err = os.WriteFile(p, []byte(a.editor.Text()), 0o600)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "mdedit: could not save unsaved changes:", err)
		return
	}
	fmt.Fprintln(os.Stderr, "mdedit: unsaved changes written to", p)
}

// recoveryPath returns a new file name in the user's cache folder.
func (a *App) recoveryPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	dir = filepath.Join(dir, "mdedit")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	stamp := time.Now().Format("20060102-150405")
	return filepath.Join(dir, fmt.Sprintf("%s-%s.md", a.baseName(), stamp)), nil
}

func (a *App) showOpenDialog() {
	a.dialog = NewFileDialog("Open Markdown file", a.dir(), "", false, mdExts, a.openChosen)
}

// save writes the document, asking for a name if it has none, and then
// performs `then`.
func (a *App) save(then action) {
	if a.path == "" {
		a.saveAs(then)
		return
	}
	a.writeFile(a.path, then)
}

func (a *App) saveAs(then action) {
	a.flow.afterSave = then
	a.dialog = NewFileDialog("Save Markdown file", a.dir(), a.baseName()+".md", true, mdExts[:1], a.saveAsChosen)
}

func (a *App) exportDialog() {
	a.dialog = NewFileDialog("Export PDF", a.dir(), a.baseName()+".pdf", true, []string{".pdf"}, a.exportChosen)
}

// openURL opens a URL with the system's default handler.
func openURL(url string) error {
	cmd := openCommand(url)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

func openCommand(url string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url)
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	return exec.Command("xdg-open", url) // Linux, the BSDs
}

// configDir is where settings.json lives; tests point it elsewhere.
var configDir = defaultConfigDir()

func defaultConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "mdedit")
}

func readSettings() ([]byte, error) {
	return os.ReadFile(filepath.Join(configDir, "settings.json"))
}

func writeSettings(data []byte) error {
	if configDir == "" {
		return os.ErrNotExist
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configDir, "settings.json"), data, 0o644)
}

// reorderIMESelection is only needed in the browser (see ime.go); tests
// switch it on.
var reorderIMESelection = false
