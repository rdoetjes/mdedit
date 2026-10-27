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
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// menuBar draws the menu titles and, when one is open, its drop-down.
type menuBar struct {
	open    int // index of the open menu, or -1
	titles  []widget.Clickable
	titleX  []int // left edge of each title, recorded during layout
	items   [][]widget.Clickable
	dismiss gesture.Click
	barH    int
}

var (
	menuHover = color.NRGBA{R: 0xe8, G: 0xf0, B: 0xfe, A: 0xff}
	menuOpen  = color.NRGBA{R: 0xdd, G: 0xe3, B: 0xea, A: 0xff}
)

func (m *menuBar) ensure(menus []*menu) {
	if len(m.titles) != len(menus) {
		m.titles = make([]widget.Clickable, len(menus))
		m.titleX = make([]int, len(menus))
		m.items = make([][]widget.Clickable, len(menus))
		for i, mn := range menus {
			m.items[i] = make([]widget.Clickable, len(mn.items))
		}
		m.open = -1
	}
}

// Layout draws the bar. The drop-down is drawn separately by Overlay so it
// appears above the rest of the window.
func (m *menuBar) Layout(gtx layout.Context, th *material.Theme, menus []*menu, enabled bool) layout.Dimensions {
	m.ensure(menus)
	for i := range m.titles {
		if m.titles[i].Clicked(gtx) && enabled {
			if m.open == i {
				m.open = -1
			} else {
				m.open = i
			}
		}
		// While a menu is open, hovering another title switches to it.
		if m.open >= 0 && m.open != i && m.titles[i].Hovered() {
			m.open = i
		}
	}
	if !enabled {
		m.open = -1
	}
	dims := layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			sz := gtx.Constraints.Min
			paint.FillShape(gtx.Ops, toolbarBg, clip.Rect{Max: sz}.Op())
			return layout.Dimensions{Size: sz}
		},
		func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			x := gtx.Dp(4)
			var children []layout.FlexChild
			children = append(children, layout.Rigid(layout.Spacer{Width: 4}.Layout))
			for i, mn := range menus {
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					m.titleX[i] = x
					d := m.titles[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Background{}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								sz := gtx.Constraints.Min
								switch {
								case m.open == i:
									paint.FillShape(gtx.Ops, menuOpen, clip.UniformRRect(image.Rectangle{Max: sz}, gtx.Dp(4)).Op(gtx.Ops))
								case m.titles[i].Hovered():
									paint.FillShape(gtx.Ops, menuHover, clip.UniformRRect(image.Rectangle{Max: sz}, gtx.Dp(4)).Op(gtx.Ops))
								}
								return layout.Dimensions{Size: sz}
							},
							func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: 4, Bottom: 4, Left: 10, Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									l := material.Body2(th, mn.title)
									l.Color = buttonFg
									return l.Layout(gtx)
								})
							})
					})
					x += d.Size.X
					return d
				}))
			}
			return layout.Inset{Top: 2, Bottom: 2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
			})
		})
	m.barH = dims.Size.Y
	return dims
}

// Overlay draws the open drop-down menu over the window and runs the chosen
// command.
func (m *menuBar) Overlay(gtx layout.Context, th *material.Theme, menus []*menu) {
	if m.open < 0 || m.open >= len(menus) {
		return
	}
	mn := menus[m.open]
	items := m.items[m.open]

	for {
		ev, ok := gtx.Event(key.Filter{Name: key.NameEscape})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			m.open = -1
			return
		}
	}
	for i, c := range mn.items {
		if c != nil && items[i].Clicked(gtx) {
			m.open = -1
			c.run(gtx)
			return
		}
	}
	// A click anywhere outside the menu closes it.
	for {
		ev, ok := m.dismiss.Update(gtx.Source)
		if !ok {
			break
		}
		if ev.Kind == gesture.KindPress {
			m.open = -1
			return
		}
	}
	// The dismiss area leaves the bar uncovered, so titles stay clickable.
	full := clip.Rect{Min: image.Pt(0, m.barH), Max: gtx.Constraints.Max}.Push(gtx.Ops)
	m.dismiss.Add(gtx.Ops)
	full.Pop()

	st := op.Offset(image.Pt(m.titleX[m.open], m.barH)).Push(gtx.Ops)
	defer st.Pop()
	gtx.Constraints.Min = image.Point{}

	// Width: wide enough for the longest label plus its shortcut.
	rec := op.Record(gtx.Ops)
	menuW := gtx.Dp(220)
	for _, c := range mn.items {
		if c == nil {
			continue
		}
		w := measureLabel(gtx, th, c.label) + measureLabel(gtx, th, c.shortcut()) + gtx.Dp(64)
		menuW = max(menuW, w)
	}
	rec.Stop()

	macro := op.Record(gtx.Ops)
	var children []layout.FlexChild
	for i, c := range mn.items {
		if c == nil {
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				h := gtx.Dp(9)
				paint.FillShape(gtx.Ops, dividerCol, clip.Rect{Min: image.Pt(0, h/2), Max: image.Pt(menuW, h/2+1)}.Op())
				return layout.Dimensions{Size: image.Pt(menuW, h)}
			}))
			continue
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return items[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = menuW
				gtx.Constraints.Max.X = menuW
				return layout.Background{}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						sz := gtx.Constraints.Min
						if items[i].Hovered() {
							paint.FillShape(gtx.Ops, menuHover, clip.Rect{Max: sz}.Op())
						}
						return layout.Dimensions{Size: sz}
					},
					func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: 5, Bottom: 5, Left: 14, Right: 14}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									l := material.Body2(th, c.label)
									l.Color = buttonFg
									return l.Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									l := material.Body2(th, c.shortcut())
									l.Color = statusFg
									return l.Layout(gtx)
								}),
							)
						})
					})
			})
		}))
	}
	dims := layout.Inset{Top: 4, Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
	call := macro.Stop()

	// Shadow, card, border, then the items.
	r := gtx.Dp(6)
	box := image.Rectangle{Max: dims.Size}
	paint.FillShape(gtx.Ops, color.NRGBA{A: 0x22}, clip.UniformRRect(box.Add(image.Pt(0, gtx.Dp(2))).Inset(-gtx.Dp(1)), r).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, clip.UniformRRect(box, r).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, dividerCol, clip.Stroke{Path: clip.UniformRRect(box, r).Path(gtx.Ops), Width: 1}.Op())
	area := clip.Rect(box).Push(gtx.Ops)
	blockPointer(gtx)
	area.Pop()
	call.Add(gtx.Ops)
}

func measureLabel(gtx layout.Context, th *material.Theme, s string) int {
	if s == "" {
		return 0
	}
	gtx.Constraints = layout.Constraints{Max: image.Pt(1e6, 1e6)}
	return material.Body2(th, s).Layout(gtx).Size.X
}

// iconButton draws a flat toolbar button with an icon.
func iconButton(gtx layout.Context, c *widget.Clickable, ic *widget.Icon) layout.Dimensions {
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				sz := gtx.Constraints.Min
				if c.Hovered() {
					paint.FillShape(gtx.Ops, menuOpen, clip.UniformRRect(image.Rectangle{Max: sz}, gtx.Dp(4)).Op(gtx.Ops))
				}
				return layout.Dimensions{Size: sz}
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(5).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					sz := gtx.Dp(unit.Dp(20))
					gtx.Constraints = layout.Exact(image.Pt(sz, sz))
					return ic.Layout(gtx, buttonFg)
				})
			})
	})
}
