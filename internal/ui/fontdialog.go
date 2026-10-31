package ui

import (
	"image"
	"strings"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"mdedit/internal/sysfont"
)

// FontDialog chooses the document font from the installed fonts.
type FontDialog struct {
	// done receives the chosen family's name, or "" if cancelled.
	done func(name string)

	th       *material.Theme
	names    []string // all choices; nil until the scan has finished
	shown    []string // those matching the filter
	selected string
	clicks   []widget.Clickable
	filter   widget.Editor
	lastF    string
	list     widget.List
	ok       widget.Clickable
	cancel   widget.Clickable
	focused  bool
}

func newFontDialog(selected string, done func(string)) *FontDialog {
	d := &FontDialog{done: done, selected: selected}
	d.list.Axis = layout.Vertical
	d.filter.SingleLine = true
	return d
}

// setFamilies gives the dialog the installed fonts, or nil while they are
// still being looked for.
func (d *FontDialog) setFamilies(fams []sysfont.Family) {
	if fams == nil {
		return
	}
	d.names = []string{defaultFontName}
	for _, f := range fams {
		d.names = append(d.names, f.Name)
	}
	d.applyFilter()
}

func (d *FontDialog) applyFilter() {
	q := strings.ToLower(d.filter.Text())
	d.lastF = q
	d.shown = d.shown[:0]
	for _, n := range d.names {
		if strings.Contains(strings.ToLower(n), q) {
			d.shown = append(d.shown, n)
		}
	}
	if len(d.clicks) < len(d.shown) {
		d.clicks = make([]widget.Clickable, len(d.names))
	}
}

func (d *FontDialog) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	d.th = th
	if d.handleInput(gtx) {
		return layout.Dimensions{}
	}
	return modal(gtx, d.card)
}

// handleInput reports whether the dialog closed.
func (d *FontDialog) handleInput(gtx layout.Context) bool {
	if !d.focused {
		gtx.Execute(key.FocusCmd{Tag: &d.filter})
		d.focused = true
	}
	if _, esc := keyPressed(gtx, key.NameEscape); esc || d.cancel.Clicked(gtx) {
		d.done("")
		return true
	}
	if d.ok.Clicked(gtx) {
		d.done(d.selected)
		return true
	}
	d.handleRows(gtx)
	if strings.ToLower(d.filter.Text()) != d.lastF {
		d.applyFilter()
	}
	return false
}

func (d *FontDialog) handleRows(gtx layout.Context) {
	for i, n := range d.shown {
		if d.clicks[i].Clicked(gtx) {
			d.selected = n
		}
	}
}

func (d *FontDialog) card(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(440))
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.UniformInset(20).Layout(gtx, d.content)
}

func (d *FontDialog) content(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.H6(d.th, "Document Font").Layout),
		vspace(8),
		layout.Rigid(d.caption("The font of the text and headings in the preview and the PDF. "+
			"Code stays monospaced.")),
		vspace(10),
		layout.Rigid(field{th: d.th, ed: &d.filter, hint: "Filter"}.layout),
		vspace(8),
		layout.Rigid(d.fontList),
		vspace(8),
		layout.Rigid(d.caption("Only fonts the PDF can embed, with Western accents, are listed.")),
		vspace(14),
		layout.Rigid(d.buttons),
	)
}

func (d *FontDialog) caption(s string) layout.Widget {
	l := material.Caption(d.th, s)
	l.Color = col.statusFg
	return l.Layout
}

// fontList is a framed, scrolling list of the fonts.
func (d *FontDialog) fontList(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Max.Y = min(gtx.Constraints.Max.Y, gtx.Dp(320))
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	return framed(gtx, d.listInside)
}

func (d *FontDialog) listInside(gtx layout.Context) layout.Dimensions {
	if d.names == nil {
		return layout.UniformInset(10).Layout(gtx, d.caption("Looking for installed fonts…"))
	}
	return material.List(d.th, &d.list).Layout(gtx, len(d.shown), d.row)
}

func (d *FontDialog) row(gtx layout.Context, i int) layout.Dimensions {
	return d.clicks[i].Layout(gtx, fontRow{d: d, i: i}.layout)
}

// fontRow is one font in the list.
type fontRow struct {
	d *FontDialog
	i int
}

func (r fontRow) layout(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.Background{}.Layout(gtx, r.background, r.label)
}

func (r fontRow) background(gtx layout.Context) layout.Dimensions {
	box := image.Rectangle{Max: gtx.Constraints.Min}
	switch {
	case r.d.shown[r.i] == r.d.selected:
		fillRect(gtx, col.highlightBg, box)
	case r.d.clicks[r.i].Hovered():
		fillRect(gtx, col.menuHover, box)
	}
	return layout.Dimensions{Size: gtx.Constraints.Min}
}

func (r fontRow) label(gtx layout.Context) layout.Dimensions {
	l := material.Body2(r.d.th, r.d.shown[r.i])
	l.Color = col.buttonFg
	l.MaxLines = 1
	return layout.Inset{Top: 5, Bottom: 5, Left: 10, Right: 10}.Layout(gtx, l.Layout)
}

func (d *FontDialog) buttons(gtx layout.Context) layout.Dimensions {
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(button{th: d.th, click: &d.cancel, label: "Cancel"}.layout),
		hspace(8),
		layout.Rigid(button{th: d.th, click: &d.ok, label: "OK", primary: true}.layout),
	)
}
