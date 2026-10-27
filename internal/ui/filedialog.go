package ui

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
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
	d.entries = d.entries[:0]
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), ".") && !d.hidden.Value {
			continue
		}
		if !e.IsDir() && !d.matches(e) {
			continue
		}
		d.entries = append(d.entries, e)
	}
	sort.SliceStable(d.entries, func(i, j int) bool {
		a, b := d.entries[i], d.entries[j]
		if isDir(dir, a) != isDir(dir, b) {
			return isDir(dir, a)
		}
		return strings.ToLower(a.Name()) < strings.ToLower(b.Name())
	})
	d.clicks = make([]widget.Clickable, len(d.entries))
	d.list.Position = layout.Position{}
}

func isDir(dir string, e os.DirEntry) bool {
	if e.IsDir() {
		return true
	}
	if e.Type()&os.ModeSymlink != 0 {
		st, err := os.Stat(filepath.Join(dir, e.Name()))
		return err == nil && st.IsDir()
	}
	return false
}

func (d *FileDialog) matches(e os.DirEntry) bool {
	if len(d.Exts) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(e.Name()))
	for _, x := range d.Exts {
		if ext == x {
			return true
		}
	}
	return false
}

// accept resolves the typed name and finishes the dialog if it names a file.
func (d *FileDialog) accept() {
	name := strings.TrimSpace(d.name.Text())
	if name == "" {
		return
	}
	if strings.HasPrefix(name, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			name = filepath.Join(home, name[2:])
		}
	}
	path := name
	if !filepath.IsAbs(path) {
		path = filepath.Join(d.dir, name)
	}
	if st, err := os.Stat(path); err == nil && st.IsDir() {
		d.chdir(path)
		d.name.SetText("")
		return
	}
	if d.Save {
		if filepath.Ext(path) == "" && d.DefExt != "" {
			path += d.DefExt
		}
		if st, err := os.Stat(filepath.Dir(path)); err != nil || !st.IsDir() {
			d.err = "folder does not exist: " + filepath.Dir(path)
			return
		}
	} else if _, err := os.Stat(path); err != nil {
		d.err = "file not found: " + path
		return
	}
	d.Done(path)
}

func (d *FileDialog) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	// Handle input.
	if !d.focused {
		gtx.Execute(key.FocusCmd{Tag: &d.name})
		d.focused = true
	}
	for {
		ev, ok := d.name.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			d.accept()
		}
	}
	for {
		ev, ok := gtx.Event(key.Filter{Name: key.NameEscape})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			d.Done("")
			return layout.Dimensions{}
		}
	}
	if d.cancel.Clicked(gtx) {
		d.Done("")
		return layout.Dimensions{}
	}
	if d.ok.Clicked(gtx) {
		d.accept()
	}
	if d.up.Clicked(gtx) {
		d.chdir(filepath.Dir(d.dir))
	}
	if d.hidden.Update(gtx) {
		d.chdir(d.dir)
	}
	// Find the clicked entry first and act on it after the loop: entering a
	// folder replaces d.entries and d.clicks, which must not happen while
	// iterating over them.
	clicked, numClicks := -1, 0
	for i := range d.clicks {
		for {
			c, ok := d.clicks[i].Update(gtx)
			if !ok {
				break
			}
			clicked, numClicks = i, c.NumClicks
		}
	}
	if clicked >= 0 && clicked < len(d.entries) {
		e := d.entries[clicked]
		if isDir(d.dir, e) {
			d.chdir(filepath.Join(d.dir, e.Name()))
		} else {
			d.name.SetText(e.Name())
			if numClicks >= 2 {
				d.accept()
			}
		}
	}

	return modal(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(680))
		gtx.Constraints.Max.Y = min(gtx.Constraints.Max.Y, gtx.Dp(520))
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.UniformInset(16).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.H6(th, d.Title)
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: 8}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return smallButton(gtx, th, &d.up, "↑ Up")
						}),
						layout.Rigid(layout.Spacer{Width: 8}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							l := material.Body2(th, d.dir)
							l.MaxLines = 1
							return l.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.CheckBox(th, &d.hidden, "Hidden").Layout(gtx)
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Height: 8}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return widget.Border{Color: borderColor, Width: 1, CornerRadius: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min = gtx.Constraints.Max
						return material.List(th, &d.list).Layout(gtx, len(d.entries), func(gtx layout.Context, i int) layout.Dimensions {
							return d.entry(gtx, th, i)
						})
					})
				}),
				layout.Rigid(layout.Spacer{Height: 8}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if d.err == "" {
						return layout.Dimensions{}
					}
					l := material.Body2(th, d.err)
					l.Color = color.NRGBA{R: 0xcf, G: 0x22, B: 0x2e, A: 0xff}
					return layout.Inset{Bottom: 8}.Layout(gtx, l.Layout)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					label := "Open"
					if d.Save {
						label = "Save"
					}
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(material.Body1(th, "Name:").Layout),
						layout.Rigid(layout.Spacer{Width: 8}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return widget.Border{Color: borderColor, Width: 1, CornerRadius: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(6).Layout(gtx, material.Editor(th, &d.name, "file name or path").Layout)
							})
						}),
						layout.Rigid(layout.Spacer{Width: 8}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return smallButton(gtx, th, &d.cancel, "Cancel")
						}),
						layout.Rigid(layout.Spacer{Width: 8}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return primaryButton(gtx, th, &d.ok, label)
						}),
					)
				}),
			)
		})
	})
}

func (d *FileDialog) entry(gtx layout.Context, th *material.Theme, i int) layout.Dimensions {
	e := d.entries[i]
	dir := isDir(d.dir, e)
	return d.clicks[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if d.clicks[i].Hovered() || (!dir && e.Name() == d.name.Text()) {
			paint.FillShape(gtx.Ops, color.NRGBA{R: 0xe8, G: 0xf0, B: 0xfe, A: 0xff},
				clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(28))}.Op())
		}
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Inset{Left: 8, Right: 8, Top: 4, Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			name := e.Name()
			l := material.Body1(th, name)
			if dir {
				l.Text = name + "/"
				l.Font.Weight = font.Bold
			}
			l.MaxLines = 1
			return l.Layout(gtx)
		})
	})
}

var borderColor = color.NRGBA{R: 0xd1, G: 0xd9, B: 0xe0, A: 0xff}

// modal draws a dimmed backdrop and centres a white card on it.
func modal(gtx layout.Context, w layout.Widget) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 0x66}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	// Swallow clicks on the backdrop.
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	blockPointer(gtx)
	area.Pop()
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = image.Point{}
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				r := gtx.Dp(8)
				paint.FillShape(gtx.Ops, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
					clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, r).Op(gtx.Ops))
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}, w)
	})
}

// ConfirmDialog asks a question with up to three answers.
type ConfirmDialog struct {
	Message string
	Buttons []string         // the last button is the default action
	Done    func(choice int) // index into Buttons, or -1 if dismissed

	clicks []widget.Clickable
}

func (c *ConfirmDialog) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if len(c.clicks) != len(c.Buttons) {
		c.clicks = make([]widget.Clickable, len(c.Buttons))
	}
	for {
		ev, ok := gtx.Event(key.Filter{Name: key.NameEscape}, key.Filter{Name: key.NameReturn})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			if e.Name == key.NameEscape {
				c.Done(-1)
			} else {
				c.Done(len(c.Buttons) - 1)
			}
			return layout.Dimensions{}
		}
	}
	for i := range c.clicks {
		if c.clicks[i].Clicked(gtx) {
			c.Done(i)
			return layout.Dimensions{}
		}
	}
	return modal(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(460))
		return layout.UniformInset(20).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(material.Body1(th, c.Message).Layout),
				layout.Rigid(layout.Spacer{Height: 16}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					var children []layout.FlexChild
					for i, b := range c.Buttons {
						if i > 0 {
							children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout))
						}
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if i == len(c.Buttons)-1 {
								return primaryButton(gtx, th, &c.clicks[i], b)
							}
							return smallButton(gtx, th, &c.clicks[i], b)
						}))
					}
					return layout.Flex{}.Layout(gtx, children...)
				}),
			)
		})
	})
}
