package ui

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// FileDialog is an in-window file chooser. It works the same on every
// platform, including FreeBSD and window managers without a desktop portal.
type FileDialog struct {
	Title  string
	Save   bool     // choose a destination instead of an existing file
	Exts   []string // extensions to show, e.g. ".md"; empty shows all
	DefExt string   // appended to typed names without an extension
	// Done is called with the chosen path, or "" if cancelled.
	Done func(path string)

	th      *material.Theme
	dir     string
	entries []os.DirEntry
	clicks  []widget.Clickable
	list    widget.List
	name    widget.Editor
	ok      widget.Clickable
	cancel  widget.Clickable
	up      widget.Clickable
	hidden  widget.Bool
	err     string
	focused bool
}

// NewFileDialog opens a dialog in dir with name pre-filled.
func NewFileDialog(title, dir, name string, save bool, exts []string, done func(string)) *FileDialog {
	d := &FileDialog{Title: title, Save: save, Exts: exts, Done: done}
	if len(exts) > 0 {
		d.DefExt = exts[0]
	}
	d.list.Axis = layout.Vertical
	d.name.SingleLine = true
	d.name.Submit = true
	d.name.SetText(name)
	d.name.SetCaret(len([]rune(name)), 0)
	if dir == "" {
		dir, _ = os.Getwd()
	}
	d.chdir(dir)
	return d
}

// ---- Directory contents ----

func (d *FileDialog) chdir(dir string) {
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		d.err = err.Error()
		return
	}
	d.err = ""
	d.dir = dir
	d.entries = d.visible(ents)
	slices.SortStableFunc(d.entries, d.compareEntries)
	d.clicks = make([]widget.Clickable, len(d.entries))
	d.list.Position = layout.Position{}
}

// visible filters out hidden files (unless shown) and files of other types.
func (d *FileDialog) visible(ents []os.DirEntry) []os.DirEntry {
	var out []os.DirEntry
	for _, e := range ents {
		if d.shows(e) {
			out = append(out, e)
		}
	}
	return out
}

func (d *FileDialog) shows(e os.DirEntry) bool {
	if strings.HasPrefix(e.Name(), ".") && !d.hidden.Value {
		return false
	}
	return e.IsDir() || d.matches(e)
}

// compareEntries sorts folders first, then by name ignoring case.
func (d *FileDialog) compareEntries(a, b os.DirEntry) int {
	ad, bd := isDir(d.dir, a), isDir(d.dir, b)
	if ad != bd {
		if ad {
			return -1
		}
		return 1
	}
	return strings.Compare(strings.ToLower(a.Name()), strings.ToLower(b.Name()))
}

func isDir(dir string, e os.DirEntry) bool {
	if e.IsDir() {
		return true
	}
	if e.Type()&os.ModeSymlink == 0 {
		return false
	}
	st, err := os.Stat(filepath.Join(dir, e.Name()))
	return err == nil && st.IsDir()
}

func (d *FileDialog) matches(e os.DirEntry) bool {
	if len(d.Exts) == 0 {
		return true
	}
	return slices.Contains(d.Exts, strings.ToLower(filepath.Ext(e.Name())))
}

// ---- Accepting a name ----

// accept resolves the typed name and finishes the dialog if it names a file.
func (d *FileDialog) accept() {
	name := strings.TrimSpace(d.name.Text())
	if name == "" {
		return
	}
	path := d.resolve(name)
	if st, err := os.Stat(path); err == nil && st.IsDir() {
		d.chdir(path)
		d.name.SetText("")
		return
	}
	if d.Save {
		d.acceptSave(path)
	} else {
		d.acceptOpen(path)
	}
}

// resolve turns a typed name into an absolute path.
func (d *FileDialog) resolve(name string) string {
	if strings.HasPrefix(name, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			name = filepath.Join(home, name[2:])
		}
	}
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(d.dir, name)
}

func (d *FileDialog) acceptSave(path string) {
	if filepath.Ext(path) == "" && d.DefExt != "" {
		path += d.DefExt
	}
	if st, err := os.Stat(filepath.Dir(path)); err != nil || !st.IsDir() {
		d.err = "folder does not exist: " + filepath.Dir(path)
		return
	}
	d.Done(path)
}

func (d *FileDialog) acceptOpen(path string) {
	if _, err := os.Stat(path); err != nil {
		d.err = "file not found: " + path
		return
	}
	d.Done(path)
}

// ---- Input ----

func (d *FileDialog) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	d.th = th
	if d.handleInput(gtx) {
		return layout.Dimensions{}
	}
	return modal(gtx, d.card)
}

// handleInput processes keys and clicks; it reports whether the dialog was
// cancelled.
func (d *FileDialog) handleInput(gtx layout.Context) bool {
	d.focusName(gtx)
	if d.cancelled(gtx) {
		d.Done("")
		return true
	}
	d.handleSubmit(gtx)
	d.handleButtons(gtx)
	d.handleEntryClicks(gtx)
	return false
}

func (d *FileDialog) focusName(gtx layout.Context) {
	if !d.focused {
		gtx.Execute(key.FocusCmd{Tag: &d.name})
		d.focused = true
	}
}

func (d *FileDialog) cancelled(gtx layout.Context) bool {
	_, esc := keyPressed(gtx, key.NameEscape)
	return esc || d.cancel.Clicked(gtx)
}

func (d *FileDialog) handleSubmit(gtx layout.Context) {
	for {
		ev, ok := d.name.Update(gtx)
		if !ok {
			return
		}
		if _, submit := ev.(widget.SubmitEvent); submit {
			d.accept()
		}
	}
}

func (d *FileDialog) handleButtons(gtx layout.Context) {
	if d.ok.Clicked(gtx) {
		d.accept()
	}
	if d.up.Clicked(gtx) {
		d.chdir(filepath.Dir(d.dir))
	}
	if d.hidden.Update(gtx) {
		d.chdir(d.dir)
	}
}

// handleEntryClicks acts on a clicked entry. It finds the click first and
// acts after: entering a folder replaces the entries being iterated over.
func (d *FileDialog) handleEntryClicks(gtx layout.Context) {
	i, n := d.clickedEntry(gtx)
	if i < 0 || i >= len(d.entries) {
		return
	}
	e := d.entries[i]
	if isDir(d.dir, e) {
		d.chdir(filepath.Join(d.dir, e.Name()))
		return
	}
	d.name.SetText(e.Name())
	if n >= 2 {
		d.accept()
	}
}

// clickedEntry returns the index of the last clicked entry and its click
// count, or -1.
func (d *FileDialog) clickedEntry(gtx layout.Context) (int, int) {
	clicked, count := -1, 0
	for i := range d.clicks {
		for {
			c, ok := d.clicks[i].Update(gtx)
			if !ok {
				break
			}
			clicked, count = i, c.NumClicks
		}
	}
	return clicked, count
}

// ---- View ----

func (d *FileDialog) card(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(680))
	gtx.Constraints.Max.Y = min(gtx.Constraints.Max.Y, gtx.Dp(520))
	gtx.Constraints.Min = gtx.Constraints.Max
	return layout.UniformInset(16).Layout(gtx, d.content)
}

func (d *FileDialog) content(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(d.title),
		vspace(8),
		layout.Rigid(d.pathBar),
		vspace(8),
		layout.Flexed(1, d.fileList),
		vspace(8),
		layout.Rigid(d.errorLine),
		layout.Rigid(d.nameBar),
	)
}

func (d *FileDialog) title(gtx layout.Context) layout.Dimensions {
	return material.H6(d.th, d.Title).Layout(gtx)
}

func (d *FileDialog) pathBar(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(button{th: d.th, click: &d.up, label: "↑ Up"}.layout),
		hspace(8),
		layout.Flexed(1, d.dirLabel),
		layout.Rigid(material.CheckBox(d.th, &d.hidden, "Hidden").Layout),
	)
}

func (d *FileDialog) dirLabel(gtx layout.Context) layout.Dimensions {
	l := material.Body2(d.th, d.dir)
	l.MaxLines = 1
	return l.Layout(gtx)
}

func (d *FileDialog) fileList(gtx layout.Context) layout.Dimensions {
	return framed(gtx, d.listView)
}

func (d *FileDialog) listView(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max
	return material.List(d.th, &d.list).Layout(gtx, len(d.entries), d.entry)
}

func (d *FileDialog) entry(gtx layout.Context, i int) layout.Dimensions {
	return d.clicks[i].Layout(gtx, fileRow{d: d, i: i}.layout)
}

func (d *FileDialog) errorLine(gtx layout.Context) layout.Dimensions {
	if d.err == "" {
		return layout.Dimensions{}
	}
	l := material.Body2(d.th, d.err)
	l.Color = errorColor
	return layout.Inset{Bottom: 8}.Layout(gtx, l.Layout)
}

func (d *FileDialog) nameBar(gtx layout.Context) layout.Dimensions {
	label := "Open"
	if d.Save {
		label = "Save"
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(material.Body1(d.th, "Name:").Layout),
		hspace(8),
		layout.Flexed(1, d.nameField),
		hspace(8),
		layout.Rigid(button{th: d.th, click: &d.cancel, label: "Cancel"}.layout),
		hspace(8),
		layout.Rigid(button{th: d.th, click: &d.ok, label: label, primary: true}.layout),
	)
}

func (d *FileDialog) nameField(gtx layout.Context) layout.Dimensions {
	return framed(gtx, d.nameEditor)
}

func (d *FileDialog) nameEditor(gtx layout.Context) layout.Dimensions {
	return layout.UniformInset(6).Layout(gtx, material.Editor(d.th, &d.name, "file name or path").Layout)
}

// fileRow draws one entry of the file list.
type fileRow struct {
	d *FileDialog
	i int
}

func (r fileRow) layout(gtx layout.Context) layout.Dimensions {
	if r.highlighted() {
		fillRect(gtx, highlightBg, image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Dp(28)))
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.Inset{Left: 8, Right: 8, Top: 4, Bottom: 4}.Layout(gtx, r.label)
}

func (r fileRow) highlighted() bool {
	e := r.d.entries[r.i]
	return r.d.clicks[r.i].Hovered() || (!isDir(r.d.dir, e) && e.Name() == r.d.name.Text())
}

func (r fileRow) label(gtx layout.Context) layout.Dimensions {
	e := r.d.entries[r.i]
	l := material.Body1(r.d.th, e.Name())
	if isDir(r.d.dir, e) {
		l.Text += "/"
		l.Font.Weight = font.Bold
	}
	l.MaxLines = 1
	return l.Layout(gtx)
}
