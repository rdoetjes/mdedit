package ui

import (
	"image/color"
	"math"
	"os/exec"
	"runtime"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
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
)

func smallButton(gtx layout.Context, th *material.Theme, c *widget.Clickable, label string) layout.Dimensions {
	b := material.Button(th, c, label)
	b.Background = buttonBg
	b.Color = buttonFg
	b.TextSize = th.TextSize * 0.85
	b.CornerRadius = 4
	b.Inset = layout.Inset{Top: 6, Bottom: 6, Left: 12, Right: 12}
	return widget.Border{Color: dividerCol, Width: 1, CornerRadius: 4}.Layout(gtx, b.Layout)
}

func primaryButton(gtx layout.Context, th *material.Theme, c *widget.Clickable, label string) layout.Dimensions {
	b := material.Button(th, c, label)
	b.Background = primaryBg
	b.TextSize = th.TextSize * 0.85
	b.CornerRadius = 4
	b.Inset = layout.Inset{Top: 7, Bottom: 7, Left: 14, Right: 14}
	return b.Layout(gtx)
}

var blockTag = new(int)

// blockPointer stops pointer events in the current clip area from reaching
// widgets below it.
func blockPointer(gtx layout.Context) {
	for {
		_, ok := gtx.Event(pointer.Filter{
			Target:  blockTag,
			Kinds:   pointer.Press | pointer.Release | pointer.Drag | pointer.Scroll,
			ScrollX: pointer.ScrollRange{Min: math.MinInt32, Max: math.MaxInt32},
			ScrollY: pointer.ScrollRange{Min: math.MinInt32, Max: math.MaxInt32},
		})
		if !ok {
			break
		}
	}
	event.Op(gtx.Ops, blockTag)
}

// openURL opens a URL with the system's default handler.
func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default: // Linux, the BSDs
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
