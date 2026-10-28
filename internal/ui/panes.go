package ui

import (
	"image"

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
	if a.dialog == nil && !gtx.Focused(&a.editor) && a.lastRow < 0 {
		gtx.Execute(key.FocusCmd{Tag: &a.editor})
	}
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

func (a *App) editorView(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max
	e := material.Editor(a.th, &a.editor, "Start writing Markdown…")
	e.Font = font.Font{Typeface: "Go Mono"}
	e.TextSize = 14
	e.LineHeightScale = 1.35
	e.SelectionColor = col.selectColor
	dims := e.Layout(gtx)
	a.find.highlight(gtx)
	return dims
}

// followCaret keeps the preview showing the block the caret is in.
func (a *App) followCaret() {
	row, _ := a.editor.CaretPos()
	if row == a.lastRow {
		return
	}
	if a.lastRow >= 0 && a.view == viewSplit {
		a.preview.EnsureVisible(a.doc.BlockAtLine(row))
	}
	a.lastRow = row
}
