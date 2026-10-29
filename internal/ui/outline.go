package ui

import (
	"image"
	"slices"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"mdedit/internal/mdmodel"
)

// outlineWidth is the width of the outline sidebar, at most a third of
// the window.
const outlineWidth = unit.Dp(240)

// outline is the sidebar listing the document's headings. Clicking one
// shows it at the top of the editor and the preview; the heading of the
// section with the caret is highlighted.
type outline struct {
	a       *App
	open    bool
	items   []outlineItem
	clicks  []widget.Clickable
	list    widget.List
	current int // index of the highlighted item, or -1
	// jump is a rune offset to show at the top of the editor on the next
	// frame, or -1.
	jump int
}

// outlineItem is one heading.
type outlineItem struct {
	level  int
	text   string
	block  int // index in the document's blocks
	offset int // rune offset of the heading's line in the text
}

func (o *outline) init(a *App, st settings) {
	o.a, o.current, o.jump = a, -1, -1
	o.list.Axis = layout.Vertical
	o.setOpen(st.Outline)
}

func (o *outline) setOpen(open bool) {
	o.open = open
	o.a.cmds.outline.label = "Show Outline"
	if open {
		o.a.cmds.outline.label = "Hide Outline"
	}
}

// update collects the headings of a newly parsed text.
func (o *outline) update(doc *mdmodel.Document, text []rune) {
	o.items = o.items[:0]
	starts := lineStarts(text)
	for i, b := range doc.Blocks {
		if b.Kind == mdmodel.Heading && b.Line < len(starts) {
			o.items = append(o.items, outlineItem{level: b.Level, text: headingText(b), block: i, offset: starts[b.Line]})
		}
	}
	for len(o.clicks) < len(o.items) {
		o.clicks = append(o.clicks, widget.Clickable{})
	}
}

// lineStarts returns the rune offset at which each line starts.
func lineStarts(text []rune) []int {
	starts := []int{0}
	for i, r := range text {
		if r == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func headingText(b mdmodel.Block) string {
	var sb strings.Builder
	for _, s := range b.Spans {
		sb.WriteString(s.Text)
	}
	if t := strings.TrimSpace(sb.String()); t != "" {
		return t
	}
	return "(empty heading)"
}

// follow highlights the heading of the section holding the caret, and
// scrolls the list to it when it changes.
func (o *outline) follow() {
	caret, _ := o.a.editor.Selection()
	i, found := slices.BinarySearchFunc(o.items, caret, itemBefore)
	if !found {
		i--
	}
	if i != o.current && i >= 0 && !o.visible(i) {
		o.list.ScrollTo(i)
	}
	o.current = i
}

func itemBefore(it outlineItem, offset int) int { return it.offset - offset }

func (o *outline) visible(i int) bool {
	p := o.list.Position
	return p.Count > 0 && i >= p.First && i < p.First+p.Count
}

// handleClicks goes to a clicked heading.
func (o *outline) handleClicks(gtx layout.Context) {
	for i := range o.items {
		if o.clicks[i].Clicked(gtx) {
			o.goTo(gtx, o.items[i])
		}
	}
}

// goTo shows a heading at the top of the editor (on the next frame) and
// of the preview, and puts the caret on it.
func (o *outline) goTo(gtx layout.Context, it outlineItem) {
	o.jump = it.offset
	o.a.editor.SetCaret(it.offset, it.offset)
	if o.a.view != viewEditor {
		o.a.preview.ScrollToBlock(it.block)
	}
	o.a.lastRow = -1 // so followCaret doesn't scroll the preview again
	o.a.invalidate()
}

// scrollEditor carries out a jump: Gio's editor scrolls just far enough to
// show the caret, which leaves a heading below the view at the bottom. So
// the editor is first laid out (and the result thrown away) with the caret
// at the end, then the caret is moved up to the heading, which then ends
// up at the top.
func (o *outline) scrollEditor(gtx layout.Context, e material.EditorStyle) {
	if o.jump < 0 {
		return
	}
	end := o.a.editor.Len()
	o.a.editor.SetCaret(end, end)
	m := op.Record(gtx.Ops)
	e.Layout(gtx)
	m.Stop()
	o.a.editor.SetCaret(o.jump, o.jump)
	o.jump = -1
}

// layout draws the sidebar.
func (o *outline) layout(gtx layout.Context) layout.Dimensions {
	o.handleClicks(gtx)
	o.follow()
	fillRect(gtx, col.toolbarBg, image.Rectangle{Max: gtx.Constraints.Max})
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(o.header),
		layout.Flexed(1, o.body),
	)
}

func (o *outline) header(gtx layout.Context) layout.Dimensions {
	l := material.Caption(o.a.th, "OUTLINE")
	l.Color = col.statusFg
	l.Font.Weight = font.SemiBold
	return layout.Inset{Top: 10, Bottom: 6, Left: 14}.Layout(gtx, l.Layout)
}

func (o *outline) body(gtx layout.Context) layout.Dimensions {
	if len(o.items) == 0 {
		l := material.Body2(o.a.th, "No headings yet")
		l.Color = col.statusFg
		return layout.Inset{Top: 6, Left: 14}.Layout(gtx, l.Layout)
	}
	return material.List(o.a.th, &o.list).Layout(gtx, len(o.items), o.row)
}

func (o *outline) row(gtx layout.Context, i int) layout.Dimensions {
	return o.clicks[i].Layout(gtx, outlineRow{o: o, i: i}.layout)
}

// outlineRow is one heading in the sidebar.
type outlineRow struct {
	o *outline
	i int
}

func (r outlineRow) layout(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.Background{}.Layout(gtx, r.background, r.label)
}

func (r outlineRow) background(gtx layout.Context) layout.Dimensions {
	switch {
	case r.i == r.o.current:
		fillRect(gtx, col.highlightBg, image.Rectangle{Max: gtx.Constraints.Min})
	case r.o.clicks[r.i].Hovered():
		fillRect(gtx, col.menuHover, image.Rectangle{Max: gtx.Constraints.Min})
	}
	return layout.Dimensions{Size: gtx.Constraints.Min}
}

// label indents the heading by its level; top-level headings are bold.
func (r outlineRow) label(gtx layout.Context) layout.Dimensions {
	it := r.o.items[r.i]
	indent := unit.Dp(14 + 12*(min(it.level, 6)-1))
	l := material.Body2(r.o.a.th, it.text)
	l.Color = col.buttonFg
	l.MaxLines = 1
	if it.level == 1 {
		l.Font.Weight = font.SemiBold
	}
	return layout.Inset{Top: 5, Bottom: 5, Left: indent, Right: 10}.Layout(gtx, l.Layout)
}

// cmdToggleOutline shows or hides the sidebar and remembers the choice.
func (a *App) cmdToggleOutline(layout.Context) {
	a.outline.setOpen(!a.outline.open)
	st := loadSettings()
	st.Outline = a.outline.open
	a.saveSettings(st, "the outline setting")
}

// mainArea lays out the outline, if shown, left of the panes.
func (a *App) mainArea(gtx layout.Context) layout.Dimensions {
	if !a.outline.open {
		return a.panes(gtx)
	}
	size := gtx.Constraints.Max
	w := min(gtx.Dp(outlineWidth), size.X/3)
	layoutAt(gtx, 0, w, a.outline.layout)
	fillRect(gtx, col.dividerCol, image.Rect(w, 0, w+1, size.Y))
	layoutAt(gtx, w+1, size.X-w-1, a.panes)
	return layout.Dimensions{Size: size}
}
