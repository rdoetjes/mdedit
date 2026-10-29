package ui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"

	"mdedit/internal/render"
)

// The editor's caret is a block: a yellow box over the character at the
// caret, which is drawn in red, so the caret is easy to find. It doesn't
// blink, and shows while the editor has the keyboard focus. Gio's own thin
// caret is switched off (Editor.HideCaret, a patch in third_party).

// editorFont is how the editor's text is shaped.
var editorFont = render.FontSpec{Mono: true}

// drawCaret draws the block caret; call it right after laying out the
// editor, with the same origin. size is the editor's visible size.
func (a *App) drawCaret(gtx layout.Context, size image.Point) {
	if !gtx.Focused(&a.editor) {
		return
	}
	caret, _ := a.editor.Selection()
	ch := caretChar(a.runes, caret)
	px := float32(gtx.Sp(editorTextSize))
	gs := a.caretText.shape(editorFont, px, ch)
	if len(gs) == 0 {
		return
	}
	pos := a.editor.CaretCoords()
	box := image.Rect(rnd(pos.X)-max(gtx.Dp(1)/2, 1), rnd(pos.Y)-gs[0].Ascent.Ceil(),
		rnd(pos.X)+advance(gs).Ceil(), rnd(pos.Y)+gs[0].Descent.Ceil())
	area := clip.Rect{Max: size}.Push(gtx.Ops)
	fillRect(gtx, col.caretBg, box)
	if ch != " " {
		a.caretText.draw(gtx, editorFont, px, ch, pos.X, pos.Y, col.caretFg)
	}
	area.Pop()
}

// caretChar returns the character at the caret, or a space at the end of
// a line or the text, where the block shows as an empty box.
func caretChar(text []rune, caret int) string {
	if caret < 0 || caret >= len(text) || text[caret] == '\n' {
		return " "
	}
	return string(text[caret])
}

// advance is the width of shaped text.
func advance(gs []text.Glyph) fixed.Int26_6 {
	var w fixed.Int26_6
	for _, g := range gs {
		w += g.Advance
	}
	return w
}
