package ui

import (
	"image"
	"image/color"
	"math"
	"os/exec"
	"runtime"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var (
	toolbarBg   = color.NRGBA{R: 0xf3, G: 0xf4, B: 0xf6, A: 0xff}
	buttonBg    = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	buttonFg    = color.NRGBA{R: 0x1f, G: 0x23, B: 0x28, A: 0xff}
	primaryBg   = color.NRGBA{R: 0x1f, G: 0x88, B: 0x3d, A: 0xff}
	editorBg    = color.NRGBA{R: 0xfb, G: 0xfb, B: 0xfc, A: 0xff}
	dividerCol  = color.NRGBA{R: 0xd1, G: 0xd9, B: 0xe0, A: 0xff}
	statusFg    = color.NRGBA{R: 0x59, G: 0x63, B: 0x6e, A: 0xff}
	selectColor = color.NRGBA{R: 0x09, G: 0x69, B: 0xda, A: 0x40}
	borderColor = color.NRGBA{R: 0xd1, G: 0xd9, B: 0xe0, A: 0xff}
	errorColor  = color.NRGBA{R: 0xcf, G: 0x22, B: 0x2e, A: 0xff}
	highlightBg = color.NRGBA{R: 0xe8, G: 0xf0, B: 0xfe, A: 0xff}
	white       = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	backdrop    = color.NRGBA{A: 0x66}
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
	return widget.Border{Color: dividerCol, Width: 1, CornerRadius: 4}.Layout(gtx, b.plainStyle().Layout)
}

func (b button) plainStyle() material.ButtonStyle {
	s := material.Button(b.th, b.click, b.label)
	s.Background = buttonBg
	s.Color = buttonFg
	s.TextSize = b.th.TextSize * 0.85
	s.CornerRadius = 4
	s.Inset = layout.Inset{Top: 6, Bottom: 6, Left: 12, Right: 12}
	return s
}

func (b button) primaryStyle() material.ButtonStyle {
	s := material.Button(b.th, b.click, b.label)
	s.Background = primaryBg
	s.TextSize = b.th.TextSize * 0.85
	s.CornerRadius = 4
	s.Inset = layout.Inset{Top: 7, Bottom: 7, Left: 14, Right: 14}
	return s
}

// Flex children for fixed gaps.
func vspace(dp unit.Dp) layout.FlexChild { return layout.Rigid(layout.Spacer{Height: dp}.Layout) }
func hspace(dp unit.Dp) layout.FlexChild { return layout.Rigid(layout.Spacer{Width: dp}.Layout) }

// framed draws w inside a thin rounded border.
func framed(gtx layout.Context, w layout.Widget) layout.Dimensions {
	return widget.Border{Color: borderColor, Width: 1, CornerRadius: 4}.Layout(gtx, w)
}

func fillRect(gtx layout.Context, col color.NRGBA, r image.Rectangle) {
	paint.FillShape(gtx.Ops, col, clip.Rect(r).Op())
}

func fillRoundRect(gtx layout.Context, col color.NRGBA, r image.Rectangle, radius int) {
	paint.FillShape(gtx.Ops, col, clip.UniformRRect(r, radius).Op(gtx.Ops))
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

// modal draws a dimmed backdrop over the window and centres w on a card.
func modal(gtx layout.Context, w layout.Widget) layout.Dimensions {
	fillRect(gtx, backdrop, image.Rectangle{Max: gtx.Constraints.Max})
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	blockPointer(gtx) // swallow clicks on the backdrop
	area.Pop()
	return layout.Center.Layout(gtx, card{w}.layout)
}

// card draws its content on a white rounded background.
type card struct{ content layout.Widget }

func (c card) layout(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min = image.Point{}
	return layout.Background{}.Layout(gtx, whiteCard, c.content)
}

func whiteCard(gtx layout.Context) layout.Dimensions {
	fillRoundRect(gtx, white, image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(8))
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

// openURL opens a URL with the system's default handler.
func openURL(url string) error {
	cmd := openCommand(url)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

func openCommand(url string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url)
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	return exec.Command("xdg-open", url) // Linux, the BSDs
}
