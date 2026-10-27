// Package ui implements the editor window.
package ui

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"mdedit/internal/mdmodel"
	"mdedit/internal/pdfout"
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

	view      viewMode
	split     float32
	drag      gesture.Drag
	dragStart float32

	btnNew, btnOpen, btnSave, btnSaveAs, btnPDF, btnView widget.Clickable

	dialog dialog
}

// New creates the editor, loading path if it is non-empty. A path that does
// not exist yet becomes the save destination of a new document.
func New(path string) (*App, error) {
	shaper := text.NewShaper(text.WithCollection(gofont.Collection()))
	th := material.NewTheme()
	th.Shaper = shaper
	th.TextSize = 15
	a := &App{th: th, split: 0.5, lastRow: -1}
	a.preview = NewPreview(shaper)
	a.preview.OnLink = a.followLink
	a.editor.WrapPolicy = text.WrapWords

	if path != "" {
		abs, err := filepath.Abs(path)
		if err == nil {
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
			return nil, err
		}
	} else {
		a.load("", "")
	}
	return a, nil
}

func (a *App) load(path, content string) {
	a.path = path
	a.saved = content
	a.editor.SetText(content)
	a.editor.SetCaret(0, 0)
	if path != "" {
		a.preview.BaseDir = filepath.Dir(path)
	} else {
		a.preview.BaseDir, _ = os.Getwd()
	}
	a.reparse()
	a.preview.ScrollToBlock(0)
	a.lastRow = -1
}

func (a *App) reparse() {
	txt := a.editor.Text()
	a.doc = mdmodel.Parse([]byte(txt))
	a.preview.SetDocument(a.doc)
	a.dirty = txt != a.saved
	a.words = countWords(txt)
}

func countWords(s string) int {
	n := 0
	in := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			in = false
		} else if !in {
			in = true
			n++
		}
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

// saveRecovery keeps unsaved work when the window is closed with changes.
func (a *App) saveRecovery() {
	if !a.dirty {
		return
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	dir = filepath.Join(dir, "mdedit")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		fmt.Fprintln(os.Stderr, "mdedit: could not save unsaved changes:", err)
		return
	}
	name := "untitled"
	if a.path != "" {
		name = strings.TrimSuffix(filepath.Base(a.path), filepath.Ext(a.path))
	}
	p := filepath.Join(dir, fmt.Sprintf("%s-%s.md", name, time.Now().Format("20060102-150405")))
	if err := os.WriteFile(p, []byte(a.editor.Text()), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "mdedit: could not save unsaved changes:", err)
		return
	}
	fmt.Fprintln(os.Stderr, "mdedit: unsaved changes written to", p)
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

// File operations.

func (a *App) dir() string {
	if a.path != "" {
		return filepath.Dir(a.path)
	}
	d, _ := os.Getwd()
	return d
}

func (a *App) baseName() string {
	if a.path == "" {
		return "untitled"
	}
	return strings.TrimSuffix(filepath.Base(a.path), filepath.Ext(a.path))
}

// confirmDiscard runs then, after asking to save unsaved changes first.
func (a *App) confirmDiscard(then func()) {
	if !a.dirty {
		then()
		return
	}
	a.dialog = &ConfirmDialog{
		Message: "The document has unsaved changes. Save them first?",
		Buttons: []string{"Cancel", "Don't save", "Save"},
		Done: func(choice int) {
			a.dialog = nil
			switch choice {
			case 1:
				a.dirty = false
				then()
			case 2:
				a.save(then)
			}
		},
	}
}

func (a *App) newDoc() {
	a.confirmDiscard(func() {
		a.load("", "")
		a.setStatus("New document")
	})
}

func (a *App) openDialog() {
	a.confirmDiscard(func() {
		a.dialog = NewFileDialog("Open Markdown file", a.dir(), "", false, mdExts, func(p string) {
			a.dialog = nil
			if p != "" {
				a.openFile(p)
			}
		})
	})
}

func (a *App) openFile(p string) {
	data, err := os.ReadFile(p)
	if err != nil {
		a.setStatus("Open failed: %v", err)
		return
	}
	a.load(p, string(data))
	a.setStatus("Opened %s", p)
}

// save writes the document, asking for a name if it has none, and runs then
// on success.
func (a *App) save(then func()) {
	if a.path == "" {
		a.saveAs(then)
		return
	}
	a.writeFile(a.path, then)
}

func (a *App) saveAs(then func()) {
	a.dialog = NewFileDialog("Save Markdown file", a.dir(), a.baseName()+".md", true, mdExts[:1], func(p string) {
		a.dialog = nil
		if p == "" {
			return
		}
		a.confirmOverwrite(p, p != a.path, func() { a.writeFile(p, then) })
	})
}

func (a *App) confirmOverwrite(p string, ask bool, then func()) {
	if _, err := os.Stat(p); err != nil || !ask {
		then()
		return
	}
	a.dialog = &ConfirmDialog{
		Message: filepath.Base(p) + " already exists. Replace it?",
		Buttons: []string{"Cancel", "Replace"},
		Done: func(choice int) {
			a.dialog = nil
			if choice == 1 {
				then()
			}
		},
	}
}

func (a *App) writeFile(p string, then func()) {
	txt := a.editor.Text()
	if err := os.WriteFile(p, []byte(txt), 0o644); err != nil {
		a.setStatus("Save failed: %v", err)
		return
	}
	a.path = p
	a.saved = txt
	a.dirty = false
	a.preview.BaseDir = filepath.Dir(p)
	a.setStatus("Saved %s", p)
	if then != nil {
		then()
	}
}

func (a *App) exportDialog() {
	a.dialog = NewFileDialog("Export PDF", a.dir(), a.baseName()+".pdf", true, []string{".pdf"}, func(p string) {
		a.dialog = nil
		if p == "" {
			return
		}
		a.confirmOverwrite(p, true, func() { a.exportPDF(p) })
	})
}

func (a *App) exportPDF(p string) {
	f, err := os.Create(p)
	if err != nil {
		a.setStatus("Export failed: %v", err)
		return
	}
	err = pdfout.Write(f, a.doc, pdfout.Options{BaseDir: a.preview.BaseDir, Title: a.baseName()})
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		a.setStatus("Export failed: %v", err)
		return
	}
	a.setStatus("Exported %s", p)
}

func (a *App) followLink(url string) {
	switch {
	case strings.HasPrefix(url, "#"):
		if i := a.doc.HeadingIndex(url[1:]); i >= 0 {
			a.preview.ScrollToBlock(i)
		} else {
			a.setStatus("No heading with anchor %s", url)
		}
	case strings.Contains(url, ":"):
		if err := openURL(url); err != nil {
			a.setStatus("Could not open %s: %v", url, err)
		}
	default:
		// A relative link: open Markdown files here, anything else externally.
		p := url
		if i := strings.IndexByte(p, '#'); i >= 0 {
			p = p[:i]
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(a.preview.BaseDir, p)
		}
		if _, err := os.Stat(p); err != nil {
			a.setStatus("Link target not found: %s", url)
			return
		}
		ext := strings.ToLower(filepath.Ext(p))
		for _, x := range mdExts {
			if ext == x {
				a.confirmDiscard(func() { a.openFile(p) })
				return
			}
		}
		if err := openURL(p); err != nil {
			a.setStatus("Could not open %s: %v", p, err)
		}
	}
}

func (a *App) quit() {
	a.confirmDiscard(func() {
		a.dirty = false
		a.win.Perform(system.ActionClose)
	})
}

// Layout.

func (a *App) handleShortcuts(gtx layout.Context) {
	sc := key.ModShortcut
	filters := []event.Filter{
		key.Filter{Name: "N", Required: sc},
		key.Filter{Name: "O", Required: sc},
		key.Filter{Name: "S", Required: sc},
		key.Filter{Name: "S", Required: sc | key.ModShift},
		key.Filter{Name: "E", Required: sc},
		key.Filter{Name: "Q", Required: sc},
		key.Filter{Name: "1", Required: sc},
		key.Filter{Name: "2", Required: sc},
		key.Filter{Name: "3", Required: sc},
	}
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		switch e.Name {
		case "N":
			a.newDoc()
		case "O":
			a.openDialog()
		case "S":
			if e.Modifiers.Contain(key.ModShift) {
				a.saveAs(nil)
			} else {
				a.save(nil)
			}
		case "E":
			a.exportDialog()
		case "Q":
			a.quit()
		case "1":
			a.view = viewSplit
		case "2":
			a.view = viewEditor
		case "3":
			a.view = viewPreview
		}
	}
}

func (a *App) layout(gtx layout.Context) {
	if a.dialog == nil {
		a.handleShortcuts(gtx)
		if a.btnNew.Clicked(gtx) {
			a.newDoc()
		}
		if a.btnOpen.Clicked(gtx) {
			a.openDialog()
		}
		if a.btnSave.Clicked(gtx) {
			a.save(nil)
		}
		if a.btnSaveAs.Clicked(gtx) {
			a.saveAs(nil)
		}
		if a.btnPDF.Clicked(gtx) {
			a.exportDialog()
		}
		if a.btnView.Clicked(gtx) {
			a.view = (a.view + 1) % 3
		}
	}
	for {
		ev, ok := a.editor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.ChangeEvent); ok {
			a.reparse()
		}
	}
	a.updateTitle()

	size := gtx.Constraints.Max
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(a.toolbar),
		layout.Flexed(1, a.body),
		layout.Rigid(a.statusBar),
	)
	if a.dialog != nil {
		gtx.Constraints = layout.Exact(size)
		a.dialog.Layout(gtx, a.th)
	}
}

func (a *App) toolbar(gtx layout.Context) layout.Dimensions {
	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			sz := gtx.Constraints.Min
			paint.FillShape(gtx.Ops, toolbarBg, clip.Rect{Max: sz}.Op())
			paint.FillShape(gtx.Ops, dividerCol, clip.Rect{Min: image.Pt(0, sz.Y-1), Max: sz}.Op())
			return layout.Dimensions{Size: sz}
		},
		func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Inset{Top: 6, Bottom: 6, Left: 8, Right: 8}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				btn := func(c *widget.Clickable, label string) layout.FlexChild {
					return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Right: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return smallButton(gtx, a.th, c, label)
						})
					})
				}
				viewLabel := [...]string{"View: Split", "View: Editor", "View: Preview"}[a.view]
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					btn(&a.btnNew, "New"),
					btn(&a.btnOpen, "Open"),
					btn(&a.btnSave, "Save"),
					btn(&a.btnSaveAs, "Save As"),
					btn(&a.btnPDF, "Export PDF"),
					btn(&a.btnView, viewLabel),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						name := "Untitled"
						if a.path != "" {
							name = a.path
						}
						if a.dirty {
							name += "  •"
						}
						l := material.Body2(a.th, name)
						l.Color = statusFg
						l.MaxLines = 1
						l.Alignment = text.End
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return l.Layout(gtx)
					}),
				)
			})
		})
}

func (a *App) statusBar(gtx layout.Context) layout.Dimensions {
	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			sz := gtx.Constraints.Min
			paint.FillShape(gtx.Ops, toolbarBg, clip.Rect{Max: sz}.Op())
			paint.FillShape(gtx.Ops, dividerCol, clip.Rect{Max: image.Pt(sz.X, 1)}.Op())
			return layout.Dimensions{Size: sz}
		},
		func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Inset{Top: 4, Bottom: 4, Left: 10, Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				msg := a.status
				if msg == "" || time.Since(a.statusTime) > 8*time.Second {
					msg = "Ctrl+O open · Ctrl+S save · Ctrl+E export PDF · Ctrl+1/2/3 split/editor/preview"
					if key.ModShortcut == key.ModCommand {
						msg = strings.ReplaceAll(msg, "Ctrl+", "⌘")
					}
				} else {
					gtx.Execute(op.InvalidateCmd{At: a.statusTime.Add(8 * time.Second)})
				}
				row, col := a.editor.CaretPos()
				pos := fmt.Sprintf("Ln %d, Col %d · %d words", row+1, col+1, a.words)
				return layout.Flex{}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						l := material.Caption(a.th, msg)
						l.Color = statusFg
						l.MaxLines = 1
						return l.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Caption(a.th, pos)
						l.Color = statusFg
						return l.Layout(gtx)
					}),
				)
			})
		})
}

func (a *App) body(gtx layout.Context) layout.Dimensions {
	size := gtx.Constraints.Max
	switch a.view {
	case viewEditor:
		return a.editorPane(gtx)
	case viewPreview:
		return a.preview.Layout(gtx, a.th)
	}

	divW := gtx.Dp(6)
	// Dragging the divider resizes the panes.
	for {
		ev, ok := a.drag.Update(gtx.Metric, gtx.Source, gesture.Horizontal)
		if !ok {
			break
		}
		switch ev.Kind {
		case pointer.Press:
			a.dragStart = ev.Position.X
		case pointer.Drag:
			a.split += (ev.Position.X - a.dragStart) / float32(size.X)
			a.split = min(max(a.split, 0.15), 0.85)
		}
	}
	leftW := int(a.split * float32(size.X))

	gtx1 := gtx
	gtx1.Constraints = layout.Exact(image.Pt(leftW, size.Y))
	a.editorPane(gtx1)

	st := op.Offset(image.Pt(leftW, 0)).Push(gtx.Ops)
	div := image.Rect(0, 0, divW, size.Y)
	paint.FillShape(gtx.Ops, toolbarBg, clip.Rect(div).Op())
	paint.FillShape(gtx.Ops, dividerCol, clip.Rect(image.Rect(divW/2, 0, divW/2+1, size.Y)).Op())
	area := clip.Rect(div).Push(gtx.Ops)
	pointer.CursorColResize.Add(gtx.Ops)
	a.drag.Add(gtx.Ops)
	area.Pop()
	st.Pop()

	st = op.Offset(image.Pt(leftW+divW, 0)).Push(gtx.Ops)
	gtx2 := gtx
	gtx2.Constraints = layout.Exact(image.Pt(size.X-leftW-divW, size.Y))
	a.preview.Layout(gtx2, a.th)
	st.Pop()
	return layout.Dimensions{Size: size}
}

func (a *App) editorPane(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, editorBg, clip.Rect{Max: gtx.Constraints.Max}.Op())
	if a.dialog == nil && !gtx.Focused(&a.editor) && a.lastRow < 0 {
		gtx.Execute(key.FocusCmd{Tag: &a.editor})
	}
	dims := layout.Inset{Top: 12, Bottom: 12, Left: 14, Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		e := material.Editor(a.th, &a.editor, "Start writing Markdown…")
		e.Font = font.Font{Typeface: "Go Mono"}
		e.TextSize = 14
		e.LineHeightScale = 1.35
		e.SelectionColor = selectColor
		return e.Layout(gtx)
	})
	// Keep the preview following the caret.
	if row, _ := a.editor.CaretPos(); row != a.lastRow {
		if a.lastRow >= 0 && a.view == viewSplit {
			a.preview.EnsureVisible(a.doc.BlockAtLine(row))
		}
		a.lastRow = row
	}
	return dims
}
