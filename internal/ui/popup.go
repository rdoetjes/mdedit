package ui

import (
	"image"
	"image/color"

	"gioui.org/gesture"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// popup is a drop-down list of commands, used for menus and the context menu.
// nil items are separators.
type popup struct {
	th      *material.Theme
	items   []*command
	clicks  []widget.Clickable
	dismiss gesture.Click
	width   int
}

var popupShadow = color.NRGBA{A: 0x22}

// Layout draws the items with the top-left corner at `at`, moved if needed
// to stay inside the window. A click outside the popup (except in
// keepClear) or Escape closes it. It returns the chosen command, if any, and
// whether the popup closed.
func (p *popup) Layout(gtx layout.Context, th *material.Theme, items []*command, at image.Point, keepClear image.Rectangle) (*command, bool) {
	p.th, p.items = th, items
	if len(p.clicks) < len(items) {
		p.clicks = make([]widget.Clickable, len(items))
	}
	if c, closed := p.handleInput(gtx); closed {
		return c, true
	}
	p.addDismissArea(gtx, keepClear.Max.Y)
	p.draw(gtx, at)
	return nil, false
}

func (p *popup) handleInput(gtx layout.Context) (*command, bool) {
	if _, esc := keyPressed(gtx, key.NameEscape); esc {
		return nil, true
	}
	if c := p.clickedItem(gtx); c != nil {
		return c, true
	}
	return nil, p.dismissed(gtx)
}

func (p *popup) clickedItem(gtx layout.Context) *command {
	for i, c := range p.items {
		if c != nil && p.clicks[i].Clicked(gtx) {
			return c
		}
	}
	return nil
}

// dismissed reports a press outside the popup.
func (p *popup) dismissed(gtx layout.Context) bool {
	for {
		ev, ok := p.dismiss.Update(gtx.Source)
		if !ok {
			return false
		}
		if ev.Kind == gesture.KindPress {
			return true
		}
	}
}

// addDismissArea catches presses on the window below y = top.
func (p *popup) addDismissArea(gtx layout.Context, top int) {
	area := clip.Rect{Min: image.Pt(0, top), Max: gtx.Constraints.Max}.Push(gtx.Ops)
	p.dismiss.Add(gtx.Ops)
	area.Pop()
}

func (p *popup) draw(gtx layout.Context, at image.Point) {
	win := gtx.Constraints.Max
	gtx.Constraints.Min = image.Point{}
	p.width = p.measureWidth(gtx)
	call, size := p.recordItems(gtx)
	st := op.Offset(clampInto(at, size, win)).Push(gtx.Ops)
	drawPopupFrame(gtx, size)
	call.Add(gtx.Ops)
	st.Pop()
}

// measureWidth is wide enough for the longest label plus its shortcut.
func (p *popup) measureWidth(gtx layout.Context) int {
	rec := op.Record(gtx.Ops)
	defer rec.Stop()
	w := gtx.Dp(220)
	for _, c := range p.items {
		if c != nil {
			w = max(w, measureLabel(gtx, p.th, c.label)+measureLabel(gtx, p.th, c.shortcut())+gtx.Dp(64+menuIconW))
		}
	}
	return w
}

func (p *popup) recordItems(gtx layout.Context) (op.CallOp, image.Point) {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: 4, Bottom: 4}.Layout(gtx, p.itemList)
	return macro.Stop(), dims.Size
}

func (p *popup) itemList(gtx layout.Context) layout.Dimensions {
	children := make([]layout.FlexChild, len(p.items))
	for i := range p.items {
		children[i] = layout.Rigid(popupRow{p: p, i: i}.layout)
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// clampInto moves a box of the given size at `at` so it fits in win.
func clampInto(at, size, win image.Point) image.Point {
	return image.Pt(max(0, min(at.X, win.X-size.X)), max(0, min(at.Y, win.Y-size.Y)))
}

// drawPopupFrame draws the shadow, card and border, and blocks pointer
// events from reaching what's below.
func drawPopupFrame(gtx layout.Context, size image.Point) {
	r := gtx.Dp(6)
	box := image.Rectangle{Max: size}
	shadow := box.Add(image.Pt(0, gtx.Dp(2))).Inset(-gtx.Dp(1))
	fillRoundRect(gtx, popupShadow, shadow, r)
	fillRoundRect(gtx, white, box, r)
	paint.FillShape(gtx.Ops, dividerCol, clip.Stroke{Path: clip.UniformRRect(box, r).Path(gtx.Ops), Width: 1}.Op())
	area := clip.Rect(box).Push(gtx.Ops)
	blockPointer(gtx)
	area.Pop()
}

func measureLabel(gtx layout.Context, th *material.Theme, s string) int {
	if s == "" {
		return 0
	}
	gtx.Constraints = layout.Constraints{Max: image.Pt(1e6, 1e6)}
	return material.Body2(th, s).Layout(gtx).Size.X
}

// popupRow is one item (or separator) of a popup.
type popupRow struct {
	p *popup
	i int
}

func (r popupRow) layout(gtx layout.Context) layout.Dimensions {
	if r.p.items[r.i] == nil {
		return r.separator(gtx)
	}
	return r.p.clicks[r.i].Layout(gtx, r.item)
}

func (r popupRow) separator(gtx layout.Context) layout.Dimensions {
	h := gtx.Dp(9)
	fillRect(gtx, dividerCol, image.Rect(0, h/2, r.p.width, h/2+1))
	return layout.Dimensions{Size: image.Pt(r.p.width, h)}
}

func (r popupRow) item(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = r.p.width
	gtx.Constraints.Max.X = r.p.width
	return layout.Background{}.Layout(gtx, r.background, r.content)
}

func (r popupRow) background(gtx layout.Context) layout.Dimensions {
	if r.p.clicks[r.i].Hovered() {
		fillRect(gtx, menuHover, image.Rectangle{Max: gtx.Constraints.Min})
	}
	return layout.Dimensions{Size: gtx.Constraints.Min}
}

func (r popupRow) content(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Top: 5, Bottom: 5, Left: 14, Right: 14}.Layout(gtx, r.labels)
}

func (r popupRow) labels(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(r.icon),
		layout.Flexed(1, r.label),
		layout.Rigid(r.shortcut),
	)
}

// menuIconW is the width of the icon column in menus, in dp.
const menuIconW = 26

// icon draws the item's icon, or leaves its space empty.
func (r popupRow) icon(gtx layout.Context) layout.Dimensions {
	size := image.Pt(gtx.Dp(menuIconW), gtx.Dp(18))
	ic := r.p.items[r.i].icon
	if ic == nil {
		return layout.Dimensions{Size: size}
	}
	gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(18), gtx.Dp(18)))
	ic.Layout(gtx, statusFg)
	return layout.Dimensions{Size: size}
}

func (r popupRow) label(gtx layout.Context) layout.Dimensions {
	l := material.Body2(r.p.th, r.p.items[r.i].label)
	l.Color = buttonFg
	return l.Layout(gtx)
}

func (r popupRow) shortcut(gtx layout.Context) layout.Dimensions {
	l := material.Body2(r.p.th, r.p.items[r.i].shortcut())
	l.Color = statusFg
	return l.Layout(gtx)
}
