package ui

import (
	"image"
	"path/filepath"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func (a *App) toolbar(gtx layout.Context) layout.Dimensions {
	a.hint = ""
	return layout.Background{}.Layout(gtx, barWithBottomLine, a.toolbarContent)
}

func barWithBottomLine(gtx layout.Context) layout.Dimensions {
	sz := gtx.Constraints.Min
	fillRect(gtx, col.toolbarBg, image.Rectangle{Max: sz})
	fillRect(gtx, col.dividerCol, image.Rect(0, sz.Y-1, sz.X, sz.Y))
	return layout.Dimensions{Size: sz}
}

// toolbarRight is the toolbar's right margin, in dp.
const toolbarRight = 10

func (a *App) toolbarContent(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	a.toolW = gtx.Constraints.Max.X
	return layout.Inset{Top: 3, Bottom: 4, Left: 8, Right: toolbarRight}.Layout(gtx, a.toolbarItems)
}

func (a *App) toolbarItems(gtx layout.Context) layout.Dimensions {
	var children []layout.FlexChild
	for _, c := range a.toolbarCmds {
		if c == nil {
			children = append(children, layout.Rigid(toolbarSeparator))
		} else {
			children = append(children, layout.Rigid(toolbarButton{a: a, c: c}.layout))
		}
	}
	children = append(children, layout.Flexed(1, a.toolbarRest))
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
}

// toolbarRest fills the toolbar after the buttons: the boxes for the
// document's font and size, lined up with the preview, then its name. The
// browser can't use installed fonts, so it only has the size.
func (a *App) toolbarRest(gtx layout.Context) layout.Dimensions {
	if a.view == viewEditor {
		a.fontBox.hide()
		return a.fileName(gtx)
	}
	x := a.toolW - gtx.Dp(toolbarRight) - gtx.Constraints.Max.X
	gap := max(a.previewLeft(gtx, a.toolW)+gtx.Dp(10)-x, gtx.Dp(8))
	children := []layout.FlexChild{layout.Rigid(pixelGap(gap).layout)}
	if systemFontsSupported {
		a.fontBox.x = x + gap
		children = append(children, layout.Rigid(a.fontBox.layout))
	}
	children = append(children, layout.Rigid(a.sizeBox.layout), layout.Flexed(1, a.fileName))
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
}

// pixelGap is an empty width, in pixels.
type pixelGap int

func (g pixelGap) layout(layout.Context) layout.Dimensions {
	return layout.Dimensions{Size: image.Pt(int(g), 0)}
}

func toolbarSeparator(gtx layout.Context) layout.Dimensions {
	w, h := gtx.Dp(17), gtx.Dp(30)
	fillRect(gtx, col.dividerCol, image.Rect(w/2, gtx.Dp(5), w/2+1, h-gtx.Dp(5)))
	return layout.Dimensions{Size: image.Pt(w, h)}
}

// documentName is the path of the document, or just its file name if the
// path doesn't fit in the given width.
func (a *App) documentName(gtx layout.Context, width int) string {
	if a.path == "" {
		return "Untitled"
	}
	macro := op.Record(gtx.Ops)
	w := measureLabel(gtx, a.th, a.path)
	macro.Stop()
	if w > width {
		return filepath.Base(a.path)
	}
	return a.path
}

// fileName shows the document path, marked when there are unsaved changes.
func (a *App) fileName(gtx layout.Context) layout.Dimensions {
	name := a.documentName(gtx, gtx.Constraints.Max.X-gtx.Dp(24))
	if a.dirty {
		name += "  •"
	}
	l := material.Body2(a.th, name)
	l.Color = col.statusFg
	l.MaxLines = 1
	l.Alignment = text.End
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return l.Layout(gtx)
}

// toolbarButton is a command's icon button; hovering it shows the command
// in the status bar.
type toolbarButton struct {
	a *App
	c *command
}

func (b toolbarButton) layout(gtx layout.Context) layout.Dimensions {
	if b.c.click.Hovered() {
		b.a.hint = b.c.tooltip()
	}
	return layout.Inset{Right: 2}.Layout(gtx, iconButton{click: &b.c.click, icon: b.c.icon}.layout)
}

// iconButton is a flat button showing an icon.
type iconButton struct {
	click *widget.Clickable
	icon  *widget.Icon
}

func (b iconButton) layout(gtx layout.Context) layout.Dimensions {
	return b.click.Layout(gtx, b.body)
}

func (b iconButton) body(gtx layout.Context) layout.Dimensions {
	return layout.Background{}.Layout(gtx, b.background, b.padded)
}

func (b iconButton) background(gtx layout.Context) layout.Dimensions {
	if b.click.Hovered() {
		fillRoundRect(gtx, col.menuOpen, image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(4))
	}
	return layout.Dimensions{Size: gtx.Constraints.Min}
}

func (b iconButton) padded(gtx layout.Context) layout.Dimensions {
	return layout.UniformInset(5).Layout(gtx, b.glyph)
}

func (b iconButton) glyph(gtx layout.Context) layout.Dimensions {
	sz := gtx.Dp(unit.Dp(20))
	gtx.Constraints = layout.Exact(image.Pt(sz, sz))
	return b.icon.Layout(gtx, col.buttonFg)
}
