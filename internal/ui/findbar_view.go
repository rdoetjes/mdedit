package ui

import (
	"image/color"
	"slices"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var (
	matchBg   = color.NRGBA{R: 0xff, G: 0xd3, B: 0x3d, A: 0x66}
	currentBg = color.NRGBA{R: 0xff, G: 0x96, B: 0x1f, A: 0x88}
)

// layout draws the bar: the search row, and the replace row if shown.
func (f *findBar) layout(gtx layout.Context) layout.Dimensions {
	f.update()
	return layout.Background{}.Layout(gtx, barWithBottomLine, f.rows)
}

func (f *findBar) rows(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.Inset{Top: 6, Bottom: 6, Left: 10, Right: 8}.Layout(gtx, f.rowStack)
}

func (f *findBar) rowStack(gtx layout.Context) layout.Dimensions {
	children := []layout.FlexChild{layout.Rigid(f.searchRow)}
	if f.replace {
		children = append(children, vspace(6), layout.Rigid(f.replaceRow))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (f *findBar) searchRow(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, field{th: f.th, ed: &f.query, hint: "Find"}.layout),
		hspace(8),
		layout.Rigid(material.CheckBox(f.th, &f.matchCase, "Match case").Layout),
		hspace(8),
		layout.Rigid(f.count),
		hspace(4),
		layout.Rigid(iconButton{click: &f.prev, icon: f.icons.up}.layout),
		layout.Rigid(iconButton{click: &f.next, icon: f.icons.down}.layout),
		layout.Rigid(iconButton{click: &f.close, icon: f.icons.close}.layout),
	)
}

func (f *findBar) replaceRow(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, field{th: f.th, ed: &f.repl, hint: "Replace with"}.layout),
		hspace(8),
		layout.Rigid(button{th: f.th, click: &f.replOne, label: "Replace"}.layout),
		hspace(6),
		layout.Rigid(button{th: f.th, click: &f.replAll, label: "Replace All"}.layout),
	)
}

// count shows "3 of 12" in a fixed width, so the buttons don't jump.
func (f *findBar) count(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Dp(80)
	l := material.Caption(f.th, f.counter())
	l.Color = statusFg
	return l.Layout(gtx)
}

// maxHighlights limits how many matches around the caret are highlighted;
// the visible ones are always among them.
const maxHighlights = 1000

// highlight marks the matches in the editor; call it right after laying
// out the editor, with the same origin.
func (f *findBar) highlight(gtx layout.Context) {
	if !f.open || len(f.matches) == 0 {
		return
	}
	n := len([]rune(f.key.q.Text))
	from, to := f.nearCaret()
	for i := from; i < to; i++ {
		col := matchBg
		if i == f.current {
			col = currentBg
		}
		f.highlightRange(gtx, f.matches[i], f.matches[i]+n, col)
	}
}

// nearCaret returns the range of match indices around the caret to
// highlight.
func (f *findBar) nearCaret() (int, int) {
	caret, _ := f.a.editor.Selection()
	i, _ := slices.BinarySearch(f.matches, caret)
	from := max(i-maxHighlights/2, 0)
	return from, min(from+maxHighlights, len(f.matches))
}

func (f *findBar) highlightRange(gtx layout.Context, start, end int, col color.NRGBA) {
	f.a.regions = f.a.editor.Regions(start, end, f.a.regions[:0])
	for _, r := range f.a.regions {
		fillRect(gtx, col, r.Bounds)
	}
}

// field is a framed single-line text field with a hint.
type field struct {
	th   *material.Theme
	ed   *widget.Editor
	hint string
}

func (fl field) layout(gtx layout.Context) layout.Dimensions {
	return framed(gtx, fl.inner)
}

func (fl field) inner(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.UniformInset(6).Layout(gtx, material.Editor(fl.th, fl.ed, fl.hint).Layout)
}
