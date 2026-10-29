package ui

import (
	"image"
	"slices"

	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/widget/material"
)

// panes lays out the editor and/or preview according to the view mode.
func (a *App) panes(gtx layout.Context) layout.Dimensions {
	switch a.view {
	case viewEditor:
		return a.editorPane(gtx)
	case viewPreview:
		return a.previewPane(gtx)
	}
	return a.splitPanes(gtx)
}

func (a *App) previewPane(gtx layout.Context) layout.Dimensions {
	return a.preview.Layout(gtx, a.th)
}

// splitPanes puts the editor and preview side by side with a draggable
// divider between them.
func (a *App) splitPanes(gtx layout.Context) layout.Dimensions {
	size := gtx.Constraints.Max
	divW := gtx.Dp(6)
	a.handleDividerDrag(gtx, size.X)
	leftW := int(a.split * float32(size.X))
	layoutAt(gtx, 0, leftW, a.editorPane)
	a.divider(gtx, leftW, divW, size.Y)
	layoutAt(gtx, leftW+divW, size.X-leftW-divW, a.previewPane)
	return layout.Dimensions{Size: size}
}

// layoutAt lays out w at x with the given width and the full height.
func layoutAt(gtx layout.Context, x, width int, w layout.Widget) {
	st := op.Offset(image.Pt(x, 0)).Push(gtx.Ops)
	gtx.Constraints = layout.Exact(image.Pt(width, gtx.Constraints.Max.Y))
	w(gtx)
	st.Pop()
}

func (a *App) handleDividerDrag(gtx layout.Context, width int) {
	for {
		ev, ok := a.drag.Update(gtx.Metric, gtx.Source, gesture.Horizontal)
		if !ok {
			return
		}
		switch ev.Kind {
		case pointer.Press:
			a.dragStart = ev.Position.X
		case pointer.Drag:
			a.split += (ev.Position.X - a.dragStart) / float32(width)
			a.split = min(max(a.split, 0.15), 0.85)
		}
	}
}

func (a *App) divider(gtx layout.Context, x, w, h int) {
	st := op.Offset(image.Pt(x, 0)).Push(gtx.Ops)
	r := image.Rect(0, 0, w, h)
	fillRect(gtx, col.toolbarBg, r)
	fillRect(gtx, col.dividerCol, image.Rect(w/2, 0, w/2+1, h))
	area := clip.Rect(r).Push(gtx.Ops)
	pointer.CursorColResize.Add(gtx.Ops)
	a.drag.Add(gtx.Ops)
	area.Pop()
	st.Pop()
}

func (a *App) editorPane(gtx layout.Context) layout.Dimensions {
	fillRect(gtx, col.editorBg, image.Rectangle{Max: gtx.Constraints.Max})
	a.takeFocus(gtx)
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(a.findBarRow),
		layout.Flexed(1, a.editorArea),
	)
	a.followCaret()
	return dims
}

// findBarRow shows the search bar above the editor while it is open.
func (a *App) findBarRow(gtx layout.Context) layout.Dimensions {
	if !a.find.open {
		return layout.Dimensions{}
	}
	return a.find.layout(gtx)
}

func (a *App) editorArea(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Top: 12, Bottom: 12, Left: 14, Right: 10}.Layout(gtx, a.editorView)
}

// editorTextSize is the size of the editor's text, in sp.
const editorTextSize = 14

func (a *App) editorView(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max
	e := material.Editor(a.th, &a.editor, "Start writing Markdown…")
	e.Font = font.Font{Typeface: "Go Mono"}
	e.TextSize = editorTextSize
	e.LineHeightScale = 1.35
	e.SelectionColor = col.selectColor
	a.scrollToTop(gtx, e)
	dims := e.Layout(gtx)
	if reorderIMESelection {
		a.orderIMESelection(gtx)
	}
	a.spell.underline(gtx, dims.Size)
	a.find.highlight(gtx)
	a.drawCaret(gtx, dims.Size)
	return dims
}

// followCaret keeps the preview showing the block the caret is in.
func (a *App) followCaret() {
	row, _ := a.caretLine()
	if row == a.lastRow {
		return
	}
	if a.lastRow >= 0 && a.view == viewSplit {
		a.preview.EnsureVisible(a.doc.BlockAtLine(row))
	}
	a.lastRow = row
}

// takeFocus gives the editor the keyboard focus when that was asked for
// (at start-up, after a click in the preview or outline), retrying until
// the editor has it.
func (a *App) takeFocus(gtx layout.Context) {
	switch {
	case !a.focusEditor || a.dialog != nil:
	case gtx.Focused(&a.editor):
		a.focusEditor = false
	default:
		gtx.Execute(key.FocusCmd{Tag: &a.editor})
	}
}

// caretLine returns the caret's line and column (from 0) in the text.
// The editor's own CaretPos counts wrapped lines on screen, not lines of
// the text, so it can't be matched with the document's blocks.
func (a *App) caretLine() (int, int) {
	caret, _ := a.editor.Selection()
	if len(a.lines) == 0 {
		return 0, caret
	}
	i, found := slices.BinarySearch(a.lines, caret)
	if !found {
		i--
	}
	i = max(i, 0)
	return i, caret - a.lines[i]
}

// showAtTop puts the caret at a rune offset, shows that line at the top of
// the editor (on the next frame) and gives the editor the keyboard focus.
// The preview isn't scrolled to follow; callers scroll it if they want.
func (a *App) showAtTop(offset int) {
	a.editor.SetCaret(offset, offset)
	a.topOffset = offset
	a.lastRow = -1 // so followCaret leaves the preview alone
	a.focusEditor = true
	a.invalidate()
}

// scrollToTop carries out showAtTop. Gio's editor scrolls just far enough
// to show the caret, which leaves a line below the view at the bottom. So
// the editor is first laid out (and the result thrown away) with the caret
// at the end, then the caret is moved up to the line, which then ends up
// at the top (unless the text ends less than a screen below it).
func (a *App) scrollToTop(gtx layout.Context, e material.EditorStyle) {
	if a.topOffset < 0 {
		return
	}
	end := a.editor.Len()
	a.editor.SetCaret(end, end)
	m := op.Record(gtx.Ops)
	e.Layout(gtx)
	m.Stop()
	a.editor.SetCaret(a.topOffset, a.topOffset)
	a.topOffset = -1
}
