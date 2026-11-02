package ui

import (
	"image"
	"math"
	"strconv"
	"strings"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"mdedit/internal/pdfout"
)

// The range of the text size, in points.
const minTextSize, maxTextSize = 1, 80

// sizeBox is the text size box in the toolbar: a number field with buttons
// to step it. It sets the base size of the PDF, in points, and the preview
// follows in proportion. A typed size counts on Enter or when the field
// loses the focus, not at each digit: on the way to 12, 1 would be a size.
type sizeBox struct {
	a        *App
	field    widget.Editor
	up, down widget.Clickable
	upIcon   *widget.Icon
	downIcon *widget.Icon
	focused  bool // the field had the keyboard focus last frame
	h        int  // the height of the field, shared by the two buttons
}

func (b *sizeBox) init(a *App, st settings) {
	b.a = a
	b.upIcon = mustIcon(icons.NavigationArrowDropUp)
	b.downIcon = mustIcon(icons.NavigationArrowDropDown)
	b.field.SingleLine = true
	b.field.Submit = true
	b.field.Filter = "0123456789.,"
	b.field.MaxLen = 5
	b.apply(savedSize(st))
}

// savedSize returns the saved text size, within the limits.
func savedSize(st settings) float32 {
	if st.TextSize == 0 {
		return pdfout.DefaultFontSize
	}
	return clampSize(st.TextSize)
}

func clampSize(v float32) float32 {
	return min(max(v, minTextSize), maxTextSize)
}

// parseSize reads a typed size; both 10.5 and 10,5 count.
func parseSize(s string) (float32, bool) {
	v, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(s), ",", "."), 32)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return clampSize(float32(v)), true
}

// stepSize returns the next whole size up (dir 1) or down (dir -1).
func stepSize(v float32, dir int) float32 {
	if dir > 0 {
		return clampSize(float32(math.Floor(float64(v))) + 1)
	}
	return clampSize(float32(math.Ceil(float64(v))) - 1)
}

func sizeText(v float32) string {
	return strconv.FormatFloat(float64(v), 'f', -1, 32)
}

// apply uses a size in the preview and for PDFs, and shows it.
func (b *sizeBox) apply(v float32) {
	b.a.textSize = v
	b.a.preview.setScale(v / pdfout.DefaultFontSize)
	b.show()
}

// show puts the current size in the field.
func (b *sizeBox) show() {
	b.field.SetText(sizeText(b.a.textSize))
}

// set changes the size and saves it.
func (b *sizeBox) set(v float32) {
	if v == b.a.textSize {
		b.show()
		return
	}
	b.apply(v)
	st := loadSettings()
	st.TextSize = v
	b.a.saveSettings(st, "the text size")
}

// commit takes the typed size, or shows the current one if it is no size.
func (b *sizeBox) commit() {
	if v, ok := parseSize(b.field.Text()); ok {
		b.set(v)
		return
	}
	b.show()
}

func (b *sizeBox) step(dir int) {
	b.commit()
	b.set(stepSize(b.a.textSize, dir))
}

func (b *sizeBox) handleInput(gtx layout.Context) {
	b.handleFocus(gtx)
	if b.up.Clicked(gtx) {
		b.step(1)
	}
	if b.down.Clicked(gtx) {
		b.step(-1)
	}
	if gtx.Focused(&b.field) {
		b.handleKeys(gtx)
	}
	b.handleField(gtx)
}

// handleFocus selects the size when the field gets the focus, so typing
// replaces it, and takes a typed size when it loses the focus.
func (b *sizeBox) handleFocus(gtx layout.Context) {
	focused := gtx.Focused(&b.field)
	switch {
	case focused && !b.focused:
		b.field.SetCaret(b.field.Len(), 0)
	case !focused && b.focused:
		b.commit()
	}
	b.focused = focused
}

// handleKeys steps with the arrow keys, before the field takes them, and
// goes back to the current size on Escape.
func (b *sizeBox) handleKeys(gtx layout.Context) {
	for {
		e, ok := b.a.editorKey(gtx,
			key.Filter{Focus: &b.field, Name: key.NameUpArrow},
			key.Filter{Focus: &b.field, Name: key.NameDownArrow})
		if !ok {
			break
		}
		if e.Name == key.NameUpArrow {
			b.step(1)
		} else {
			b.step(-1)
		}
	}
	if _, esc := keyPressed(gtx, key.NameEscape); esc {
		b.show()
		b.a.focusEditor = true
	}
}

// handleField takes the size on Enter and returns to the editor.
func (b *sizeBox) handleField(gtx layout.Context) {
	for {
		ev, ok := b.field.Update(gtx)
		if !ok {
			return
		}
		if _, submit := ev.(widget.SubmitEvent); submit {
			b.commit()
			b.a.focusEditor = true
		}
	}
}

func (b *sizeBox) layout(gtx layout.Context) layout.Dimensions {
	b.handleInput(gtx)
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(b.box),
		hspace(4),
		layout.Rigid(b.unit),
		hspace(12),
	)
}

func (b *sizeBox) box(gtx layout.Context) layout.Dimensions {
	return framed(gtx, b.inside)
}

func (b *sizeBox) inside(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(b.fieldView),
		layout.Rigid(b.buttons),
	)
}

func (b *sizeBox) fieldView(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Dp(36)
	gtx.Constraints.Max.X = gtx.Dp(36)
	e := material.Editor(b.a.th, &b.field, "")
	e.TextSize = b.a.th.TextSize * 0.85
	dims := layout.Inset{Top: 6, Bottom: 6, Left: 6}.Layout(gtx, e.Layout)
	b.h = dims.Size.Y
	return dims
}

func (b *sizeBox) buttons(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(stepButton{b: b, click: &b.up, icon: b.upIcon}.layout),
		layout.Rigid(stepButton{b: b, click: &b.down, icon: b.downIcon}.layout),
	)
}

func (b *sizeBox) unit(gtx layout.Context) layout.Dimensions {
	l := material.Caption(b.a.th, "pt")
	l.Color = col.statusFg
	return l.Layout(gtx)
}

// stepButton is one of the small arrows, half the field's height.
type stepButton struct {
	b     *sizeBox
	click *widget.Clickable
	icon  *widget.Icon
}

func (s stepButton) layout(gtx layout.Context) layout.Dimensions {
	return s.click.Layout(gtx, s.body)
}

func (s stepButton) body(gtx layout.Context) layout.Dimensions {
	size := image.Pt(gtx.Dp(18), s.b.h/2)
	if s.click.Hovered() {
		fillRect(gtx, col.menuOpen, image.Rectangle{Max: size})
	}
	gtx.Constraints = layout.Exact(size)
	layout.Center.Layout(gtx, s.glyph)
	return layout.Dimensions{Size: size}
}

func (s stepButton) glyph(gtx layout.Context) layout.Dimensions {
	side := gtx.Dp(16)
	gtx.Constraints = layout.Exact(image.Pt(side, side))
	return s.icon.Layout(gtx, col.buttonFg)
}
