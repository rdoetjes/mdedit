package ui

import (
	"image"
	"strings"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"mdedit/internal/sysfont"
)

// fontCombo is the "PDF font" combo box in the toolbar: a text field
// showing the chosen font, with a button that drops down the list of
// fonts. Typing filters the list; clicking an entry (or Enter) chooses it.
// Escape or a click elsewhere goes back to the chosen font.
type fontCombo struct {
	a       *App
	field   widget.Editor
	arrow   widget.Clickable
	icon    *widget.Icon
	list    widget.List
	clicks  []widget.Clickable
	dismiss gesture.Click
	press   gesture.Click // on the part of the box around the text

	open    bool
	focused bool     // the field had the keyboard focus last frame
	names   []string // all fonts; nil until the scan has finished
	shown   []string // those matching the text in the field
	query   string   // the text shown was filtered by
	hot     int      // index in shown of the highlighted entry
	// box is the field and button, and dropped the list below it, as last
	// drawn, relative to the combo box; x is where that is in the window.
	box, dropped image.Rectangle
	x            int
	arrowW       int
}

func (c *fontCombo) init(a *App) {
	c.a = a
	c.icon = mustIcon(icons.NavigationArrowDropDown)
	c.list.Axis = layout.Vertical
	c.field.SingleLine = true
	c.field.Submit = true
	c.field.SetText(a.fontName())
	c.query = a.fontName()
}

// setFamilies gives the box the installed fonts.
func (c *fontCombo) setFamilies(fams []sysfont.Family) {
	c.names = []string{defaultFontName}
	for _, f := range fams {
		c.names = append(c.names, f.Name)
	}
	c.clicks = make([]widget.Clickable, len(c.names))
	c.filter(c.field.Text())
}

// filter shows the fonts holding the text; the chosen font's own name, as
// the field shows it before any typing, shows them all.
func (c *fontCombo) filter(text string) {
	c.query = text
	q := strings.ToLower(text)
	if text == c.a.fontName() {
		q = ""
	}
	c.shown = c.shown[:0]
	for _, n := range c.names {
		if strings.Contains(strings.ToLower(n), q) {
			c.shown = append(c.shown, n)
		}
	}
	c.hot = max(c.indexOf(c.a.fontName()), 0)
	c.list.ScrollTo(c.hot)
}

func (c *fontCombo) indexOf(name string) int {
	for i, n := range c.shown {
		if n == name {
			return i
		}
	}
	return -1
}

// drop opens the list, with the field's text selected so typing replaces
// it, and looks for the installed fonts the first time.
func (c *fontCombo) drop() {
	c.open = true
	c.a.scanFonts()
	c.field.SetCaret(c.field.Len(), 0)
	c.filter(c.field.Text())
}

// shut closes the list; the field shows the chosen font again.
func (c *fontCombo) shut() {
	c.open = false
	c.field.SetText(c.a.fontName())
	c.query = c.a.fontName()
}

// hide closes the list when the box goes away with the preview.
func (c *fontCombo) hide() {
	if c.open {
		c.shut()
		c.a.focusEditor = true
	}
}

// choose makes name the font and hands the keyboard back to the editor.
func (c *fontCombo) choose(name string) {
	c.a.fontChosen(name)
	c.shut()
	c.a.focusEditor = true
}

// handleInput runs before the box is laid out.
func (c *fontCombo) handleInput(gtx C) {
	c.handleFocus(gtx)
	c.handlePress(gtx)
	if c.arrow.Clicked(gtx) {
		c.toggle(gtx)
	}
	if c.open {
		c.handleArrowKeys(gtx) // before the field takes them
	}
	c.handleField(gtx)
	if c.open {
		c.handleList(gtx)
	}
}

// handleFocus drops the list when the field gets the focus, and closes it
// when something else takes the focus.
func (c *fontCombo) handleFocus(gtx C) {
	focused := gtx.Focused(&c.field)
	switch {
	case focused && !c.focused && !c.open:
		c.drop()
	case !focused && c.focused && c.open:
		c.shut()
	}
	c.focused = focused
}

// handlePress gives the field the focus on a press anywhere around its
// text: Gio's field is only as tall as its line, the box is taller.
func (c *fontCombo) handlePress(gtx C) {
	for {
		ev, ok := c.press.Update(gtx.Source)
		if !ok {
			return
		}
		if ev.Kind == gesture.KindPress {
			gtx.Execute(key.FocusCmd{Tag: &c.field})
		}
	}
}

func (c *fontCombo) toggle(gtx C) {
	if c.open {
		c.shut()
		c.a.focusEditor = true
		return
	}
	gtx.Execute(key.FocusCmd{Tag: &c.field})
}

// handleField follows the text in the field; Enter chooses.
func (c *fontCombo) handleField(gtx C) {
	for {
		ev, ok := c.field.Update(gtx)
		if !ok {
			break
		}
		if _, submit := ev.(widget.SubmitEvent); submit && c.open && len(c.shown) > 0 {
			c.choose(c.shown[c.hot])
			return
		}
	}
	if t := c.field.Text(); t != c.query {
		c.filter(t)
		c.open = c.open || gtx.Focused(&c.field)
	}
}

// handleList handles keys and clicks while the list is open.
func (c *fontCombo) handleList(gtx C) {
	if _, esc := keyPressed(gtx, key.NameEscape); esc {
		c.shut()
		c.a.focusEditor = true
		return
	}
	for i, n := range c.shown {
		if c.clicks[i].Clicked(gtx) {
			c.choose(n)
			return
		}
	}
	c.handleDismiss(gtx)
}

func (c *fontCombo) handleArrowKeys(gtx C) {
	for {
		e, ok := c.a.editorKey(gtx,
			key.Filter{Focus: &c.field, Name: key.NameUpArrow},
			key.Filter{Focus: &c.field, Name: key.NameDownArrow})
		if !ok {
			return
		}
		step := 1
		if e.Name == key.NameUpArrow {
			step = -1
		}
		c.hot = max(min(c.hot+step, len(c.shown)-1), 0)
		c.list.ScrollTo(c.hot)
	}
}

// handleDismiss closes the list on a press outside the box and the list.
func (c *fontCombo) handleDismiss(gtx C) {
	for {
		ev, ok := c.dismiss.Update(gtx.Source)
		if !ok {
			return
		}
		at := ev.Position
		if ev.Kind == gesture.KindPress && !at.In(c.box) && !at.In(c.dropped) {
			c.shut()
		}
	}
}

// layout draws the box; the list goes over the window.
func (c *fontCombo) layout(gtx C) D {
	c.handleInput(gtx)
	dims := c.row(gtx)
	if c.open {
		macro := op.Record(gtx.Ops)
		c.drawDropped(gtx)
		op.Defer(gtx.Ops, macro.Stop())
	}
	return dims
}

func (c *fontCombo) row(gtx C) D {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(c.label),
		hspace(8),
		layout.Rigid(c.combo),
		hspace(12),
	)
}

func (c *fontCombo) label(gtx C) D {
	l := material.Caption(c.a.th, "PDF font")
	l.Color = col.statusFg
	dims := l.Layout(gtx)
	c.box.Min.X = dims.Size.X + gtx.Dp(8)
	return dims
}

// combo is the framed field and button.
func (c *fontCombo) combo(gtx C) D {
	gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(260))
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	dims := framed(gtx, c.inside)
	c.box.Min.Y = 0
	c.box.Max = c.box.Min.Add(dims.Size)
	area := clip.Rect{Max: image.Pt(dims.Size.X-c.arrowW, dims.Size.Y)}.Push(gtx.Ops)
	pass := pointer.PassOp{}.Push(gtx.Ops)
	c.press.Add(gtx.Ops)
	pass.Pop()
	area.Pop()
	return dims
}

func (c *fontCombo) inside(gtx C) D {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, c.fieldView),
		layout.Rigid(func(gtx C) D {
			dims := iconButton{click: &c.arrow, icon: c.icon}.layout(gtx)
			c.arrowW = dims.Size.X
			return dims
		}),
	)
}

func (c *fontCombo) fieldView(gtx C) D {
	e := material.Editor(c.a.th, &c.field, "")
	e.TextSize = c.a.th.TextSize * 0.85
	return layout.Inset{Left: 6, Right: 4}.Layout(gtx, e.Layout)
}

// drawDropped draws the list below the box, over the rest of the window,
// and listens for presses elsewhere; they go through to what's there.
func (c *fontCombo) drawDropped(gtx C) {
	c.listenOutside(gtx)
	gtx.Constraints = layout.Exact(image.Pt(c.box.Dx(), gtx.Dp(320)))
	gtx.Constraints.Min.Y = 0
	macro := op.Record(gtx.Ops)
	dims := c.listBody(gtx)
	call := macro.Stop()
	at := image.Pt(c.box.Min.X, c.box.Max.Y)
	c.dropped = image.Rectangle{Min: at, Max: at.Add(dims.Size)}
	st := op.Offset(at).Push(gtx.Ops)
	drawPopupFrame(gtx, dims.Size)
	call.Add(gtx.Ops)
	st.Pop()
}

func (c *fontCombo) listenOutside(gtx C) {
	const far = 1 << 20
	area := clip.Rect{Min: image.Pt(-far, -far), Max: image.Pt(far, far)}.Push(gtx.Ops)
	pass := pointer.PassOp{}.Push(gtx.Ops)
	c.dismiss.Add(gtx.Ops)
	event.Op(gtx.Ops, &c.dismiss)
	pass.Pop()
	area.Pop()
}

func (c *fontCombo) listBody(gtx C) D {
	if c.names == nil {
		return c.message(gtx, "Looking for installed fonts…")
	}
	if len(c.shown) == 0 {
		return c.message(gtx, "No matching font")
	}
	return material.List(c.a.th, &c.list).Layout(gtx, len(c.shown), c.entry)
}

func (c *fontCombo) message(gtx C, s string) D {
	l := material.Body2(c.a.th, s)
	l.Color = col.statusFg
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.UniformInset(8).Layout(gtx, l.Layout)
}

// entry is one font in the list.
func (c *fontCombo) entry(gtx C, i int) D {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return c.clicks[i].Layout(gtx, func(gtx C) D {
		return layout.Background{}.Layout(gtx, c.entryBackground(i), c.entryLabel(i))
	})
}

func (c *fontCombo) entryBackground(i int) layout.Widget {
	return func(gtx C) D {
		if i == c.hot || c.clicks[i].Hovered() {
			fillRect(gtx, col.highlightBg, image.Rectangle{Max: gtx.Constraints.Min})
		}
		return D{Size: gtx.Constraints.Min}
	}
}

func (c *fontCombo) entryLabel(i int) layout.Widget {
	return func(gtx C) D {
		l := material.Body2(c.a.th, c.shown[i])
		l.Color, l.MaxLines = col.buttonFg, 1
		return layout.Inset{Top: 4, Bottom: 4, Left: 8, Right: 8}.Layout(gtx, l.Layout)
	}
}
