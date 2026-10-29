// Package ui implements the editor window.
package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"mdedit/internal/edit"
	"mdedit/internal/mdmodel"
)

type viewMode int

const (
	viewSplit viewMode = iota
	viewEditor
	viewPreview
)

var mdExts = []string{".md", ".markdown", ".mdown", ".mkd", ".txt"}

type dialog interface {
	Layout(gtx layout.Context, th *material.Theme) layout.Dimensions
}

// App is the editor window state.
type App struct {
	win     *app.Window
	th      *material.Theme
	editor  widget.Editor
	preview *Preview
	doc     *mdmodel.Document

	path    string
	saved   string // contents as last loaded or saved
	dirty   bool
	title   string
	words   int
	lastRow int

	status     string
	statusTime time.Time

	dark      bool
	view      viewMode
	split     float32
	drag      gesture.Drag
	dragStart float32

	cmds        commands
	menus       []*menu
	toolbarCmds []*command
	shortcuts   []*command
	bar         menuBar
	barH, toolH int
	hint        string // status bar text while hovering a toolbar button
	regions     []widget.Region
	ctx         contextMenu
	find        findBar
	spell       spellChecker
	outline     outline
	recent      []string // recently used files, newest first
	version     int      // counts text changes, to know when to search again

	dialog dialog
	flow   flow
	// openRequest is a file the operating system asked to open (Finder's
	// "Open With"); it arrives on another goroutine.
	openMu      sync.Mutex
	openRequest string

	// pendingEdit and pendingCmd run on the next frame; they are requested
	// by dialogs, which have no layout context.
	pendingEdit editFunc
	pendingCmd  *command

	platform platformState // desktop or browser specifics (platform_*.go)
}

// Version is shown in the About box.
var Version = "dev"

// New creates the editor, loading path if it is non-empty. A path that does
// not exist yet becomes the save destination of a new document.
func New(path string) (*App, error) {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(append(gofont.Collection(), platformFonts()...)))
	th.TextSize = 15
	a := &App{th: th, split: 0.5, lastRow: -1}
	a.buildCommands()
	a.preview = NewPreview(th.Shaper)
	a.preview.OnLink = a.followLink
	a.preview.Invalidate = a.invalidate
	a.editor.WrapPolicy = text.WrapWords
	a.editor.PaintSelectionWhenUnfocused = true // keep the found match visible
	a.find.init(a)
	st := loadSettings()
	a.setDark(startDark(st))
	a.spell.init(a, st)
	a.outline.init(a, st)
	a.recent = st.Recent
	if err := a.loadInitial(path); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *App) load(path, content string) {
	// Gio can't display tab characters; expand them as CommonMark would.
	if strings.ContainsRune(content, '\t') {
		content = edit.ExpandTabs(edit.New(content, 0, 0)).String()
		a.setStatus("Tabs converted to spaces (the file changes only when you save)")
	}
	a.path = path
	a.rememberRecent(path)
	a.saved = content
	a.editor.SetText(content)
	a.editor.SetCaret(0, 0)
	a.preview.BaseDir = a.dir()
	a.reparse()
	a.preview.ScrollToBlock(0)
	a.lastRow = -1
}

func (a *App) reparse() {
	txt := a.editor.Text()
	a.version++
	a.doc = mdmodel.Parse([]byte(txt))
	a.preview.SetDocument(a.doc)
	runes := []rune(txt)
	a.spell.check(runes, a.doc)
	a.outline.update(a.doc, runes)
	a.dirty = txt != a.saved
	a.words = countWords(txt)
	a.autosave()
}

func countWords(s string) int {
	n := 0
	inWord := false
	for _, r := range s {
		space := unicode.IsSpace(r)
		if !space && !inWord {
			n++
		}
		inWord = !space
	}
	return n
}

func (a *App) setStatus(format string, args ...any) {
	a.status = fmt.Sprintf(format, args...)
	a.statusTime = time.Now()
}

// Run drives the window until it is closed.
func (a *App) Run(w *app.Window) error {
	a.win = w
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			a.saveRecovery()
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			a.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (a *App) updateTitle() {
	name := "Untitled"
	if a.path != "" {
		name = filepath.Base(a.path)
	}
	t := name + " — mdedit"
	if a.dirty {
		t = "• " + t
	}
	if t != a.title && a.win != nil {
		a.title = t
		a.win.Option(app.Title(t))
	}
}

// invalidate asks the window for a new frame.
func (a *App) invalidate() {
	if a.win != nil {
		a.win.Invalidate()
	}
}

// dir is the folder of the document, or else the working directory —
// unless that is the root, as for apps started from the Finder, in which
// case it is the home folder.
func (a *App) dir() string {
	if a.path != "" {
		return filepath.Dir(a.path)
	}
	d, err := os.Getwd()
	if err != nil || d == "/" {
		if home, herr := os.UserHomeDir(); herr == nil {
			return home
		}
	}
	return d
}

// RequestOpen asks the editor to open a file, as the Finder does for "Open
// With". It is safe to call from any goroutine; the file is opened on the
// next frame, after asking about unsaved changes.
func (a *App) RequestOpen(path string) {
	a.openMu.Lock()
	a.openRequest = path
	a.openMu.Unlock()
	a.invalidate()
}

// takeOpenRequest returns and clears a pending open request.
func (a *App) takeOpenRequest() string {
	a.openMu.Lock()
	defer a.openMu.Unlock()
	p := a.openRequest
	a.openRequest = ""
	return p
}

func (a *App) baseName() string {
	if a.path == "" {
		return "untitled"
	}
	return strings.TrimSuffix(filepath.Base(a.path), filepath.Ext(a.path))
}
