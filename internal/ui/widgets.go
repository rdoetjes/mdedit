package ui

import (
	"image"
	"image/color"
	"math"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// C and D are the usual short names in Gio code, for short closures.
type (
	C = layout.Context
	D = layout.Dimensions
)

// button is a push button that can be used as a layout.Widget through its
// layout method.
type button struct {
	th      *material.Theme
	click   *widget.Clickable
	label   string
	primary bool
}

func (b button) layout(gtx layout.Context) layout.Dimensions {
	if b.primary {
		return b.primaryStyle().Layout(gtx)
	}
	return widget.Border{Color: col.dividerCol, Width: 1, CornerRadius: 4}.Layout(gtx, b.plainStyle().Layout)
}

func (b button) plainStyle() material.ButtonStyle {
	s := material.Button(b.th, b.click, b.label)
	s.Background = col.buttonBg
	s.Color = col.buttonFg
	s.TextSize = b.th.TextSize * 0.85
	s.CornerRadius = 4
	s.Inset = layout.Inset{Top: 6, Bottom: 6, Left: 12, Right: 12}
	return s
}

func (b button) primaryStyle() material.ButtonStyle {
	s := material.Button(b.th, b.click, b.label)
	s.Background = col.primaryBg
	s.TextSize = b.th.TextSize * 0.85
	s.CornerRadius = 4
	s.Inset = layout.Inset{Top: 7, Bottom: 7, Left: 14, Right: 14}
	return s
}

// Flex children for fixed gaps.
func vspace(dp unit.Dp) layout.FlexChild { return layout.Rigid(layout.Spacer{Height: dp}.Layout) }
func hspace(dp unit.Dp) layout.FlexChild { return layout.Rigid(layout.Spacer{Width: dp}.Layout) }

// rowBackground marks a row of a list that is selected or under the pointer.
func rowBackground(selected bool, click *widget.Clickable) layout.Widget {
	return func(gtx C) D {
		if selected || click.Hovered() {
			fillRect(gtx, col.highlightBg, image.Rectangle{Max: gtx.Constraints.Min})
		}
		return D{Size: gtx.Constraints.Min}
	}
}

// hoverFill marks a button while the pointer is over it.
func hoverFill(gtx C, click *widget.Clickable, size image.Point) {
	if click.Hovered() {
		fillRect(gtx, col.menuOpen, image.Rectangle{Max: size})
	}
}

// centredIcon draws an icon of side pixels in the middle of a box of size.
func centredIcon(gtx C, ic *widget.Icon, side int, size image.Point) {
	gtx.Constraints = layout.Exact(image.Pt(side, side))
	defer op.Offset(size.Sub(image.Pt(side, side)).Div(2)).Push(gtx.Ops).Pop()
	ic.Layout(gtx, col.buttonFg)
}

// framed draws w inside a thin rounded border.
func framed(gtx layout.Context, w layout.Widget) layout.Dimensions {
	return widget.Border{Color: col.borderColor, Width: 1, CornerRadius: 4}.Layout(gtx, w)
}

func fillRect(gtx layout.Context, c color.NRGBA, r image.Rectangle) {
	paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
}

func fillRoundRect(gtx layout.Context, c color.NRGBA, r image.Rectangle, radius int) {
	paint.FillShape(gtx.Ops, c, clip.UniformRRect(r, radius).Op(gtx.Ops))
}

// keyPressed drains global key events for the given keys and reports the
// first one pressed.
func keyPressed(gtx layout.Context, names ...key.Name) (key.Name, bool) {
	filters := make([]event.Filter, len(names))
	for i, n := range names {
		filters[i] = key.Filter{Name: n}
	}
	var pressed key.Name
	found := false
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			return pressed, found
		}
		if e, isKey := ev.(key.Event); isKey && e.State == key.Press && !found {
			pressed, found = e.Name, true
		}
	}
}

// modal draws a dimmed col.backdrop over the window and centres w on a card.
func modal(gtx layout.Context, w layout.Widget) layout.Dimensions {
	fillRect(gtx, col.backdrop, image.Rectangle{Max: gtx.Constraints.Max})
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	blockPointer(gtx) // swallow clicks on the col.backdrop
	area.Pop()
	return layout.Center.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min = image.Point{}
		return layout.Background{}.Layout(gtx, whiteCard, w)
	})
}

func whiteCard(gtx layout.Context) layout.Dimensions {
	fillRoundRect(gtx, col.cardBg, image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(8))
	return layout.Dimensions{Size: gtx.Constraints.Min}
}

var blockTag = new(int)

// blockPointer stops pointer events in the current clip area from reaching
// widgets below it.
func blockPointer(gtx layout.Context) {
	for {
		if _, ok := gtx.Event(blockFilter()); !ok {
			break
		}
	}
	event.Op(gtx.Ops, blockTag)
}

func blockFilter() pointer.Filter {
	return pointer.Filter{
		Target:  blockTag,
		Kinds:   pointer.Press | pointer.Release | pointer.Drag | pointer.Scroll,
		ScrollX: pointer.ScrollRange{Min: math.MinInt32, Max: math.MaxInt32},
		ScrollY: pointer.ScrollRange{Min: math.MinInt32, Max: math.MaxInt32},
	}
}
