package ui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// menuBar draws the menu titles and, when one is open, its drop-down.
type menuBar struct {
	th     *material.Theme
	menus  []*menu
	open   int // index of the open menu, or -1
	titles []widget.Clickable
	titleX []int // left edge of each title, recorded during layout
	titleW []int // width of each title
	popup  popup
	barH   int
}

func (m *menuBar) ensure() {
	if len(m.titles) != len(m.menus) {
		m.titles = make([]widget.Clickable, len(m.menus))
		m.titleX = make([]int, len(m.menus))
		m.titleW = make([]int, len(m.menus))
		m.open = -1
	}
}

// Layout draws the bar. The drop-down is drawn separately by Overlay so it
// appears above the rest of the window.
func (m *menuBar) Layout(gtx layout.Context, th *material.Theme, menus []*menu, enabled bool) layout.Dimensions {
	m.th, m.menus = th, menus
	m.ensure()
	m.handleClicks(gtx, enabled)
	dims := layout.Background{}.Layout(gtx, barBackground, m.titleRow)
	m.barH = dims.Size.Y
	return dims
}

func (m *menuBar) handleClicks(gtx layout.Context, enabled bool) {
	for i := range m.titles {
		if m.titles[i].Clicked(gtx) && enabled {
			m.toggle(i)
		}
		// While a menu is open, hovering another title switches to it.
		if m.open >= 0 && m.open != i && m.titles[i].Hovered() {
			m.open = i
		}
	}
	if !enabled {
		m.open = -1
	}
}

func (m *menuBar) toggle(i int) {
	if m.open == i {
		m.open = -1
	} else {
		m.open = i
	}
}

func barBackground(gtx layout.Context) layout.Dimensions {
	fillRect(gtx, col.toolbarBg, image.Rectangle{Max: gtx.Constraints.Min})
	return layout.Dimensions{Size: gtx.Constraints.Min}
}

func (m *menuBar) titleRow(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.Inset{Top: 2, Bottom: 2, Left: 4}.Layout(gtx, m.titleFlex)
}

func (m *menuBar) titleFlex(gtx layout.Context) layout.Dimensions {
	children := make([]layout.FlexChild, len(m.menus))
	for i := range m.menus {
		children[i] = layout.Rigid(func(gtx C) D { return m.title(gtx, i) })
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
}

// Overlay draws the open drop-down menu over the window and runs the chosen
// command.
func (m *menuBar) Overlay(gtx layout.Context, th *material.Theme, menus []*menu) {
	if m.open < 0 || m.open >= len(menus) {
		return
	}
	// Leave the bar uncovered so its titles stay clickable.
	bar := image.Rect(0, 0, gtx.Constraints.Max.X, m.barH)
	at := image.Pt(m.titleX[m.open], m.barH)
	chosen, closed := m.popup.Layout(gtx, th, menus[m.open].list(), at, bar)
	if closed {
		m.open = -1
	}
	if chosen != nil {
		chosen.run(gtx)
	}
}

// title lays out one title of the bar and records where it is.
func (m *menuBar) title(gtx C, i int) D {
	m.titleX[i] = gtx.Dp(4)
	for _, w := range m.titleW[:i] {
		m.titleX[i] += w
	}
	l := material.Body2(m.th, m.menus[i].title)
	l.Color = col.buttonFg
	background := func(gtx C) D { return m.titleBackground(gtx, i) }
	label := func(gtx C) D { return layout.Inset{Top: 4, Bottom: 4, Left: 10, Right: 10}.Layout(gtx, l.Layout) }
	d := m.titles[i].Layout(gtx, func(gtx C) D {
		return layout.Background{}.Layout(gtx, background, label)
	})
	m.titleW[i] = d.Size.X
	return d
}

func (m *menuBar) titleBackground(gtx C, i int) D {
	r := image.Rectangle{Max: gtx.Constraints.Min}
	switch {
	case m.open == i:
		fillRoundRect(gtx, col.menuOpen, r, gtx.Dp(4))
	case m.titles[i].Hovered():
		fillRoundRect(gtx, col.menuHover, r, gtx.Dp(4))
	}
	return D{Size: gtx.Constraints.Min}
}
