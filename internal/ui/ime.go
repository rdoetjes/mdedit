package ui

import (
	"gioui.org/io/key"
	"gioui.org/layout"
)

// orderIMESelection works around a bug in Gio's browser port. Its textarea
// "input" handler (gioui.org/app/os_js.go) assumes the editor's IME
// selection has Start <= End, but the editor reports it as (caret, anchor),
// so after selecting a word left to right Start is its right end. Typing
// then replaced the word correctly but put the caret where the word used to
// end, and the next characters went in further to the right.
//
// Call it right after laying out the editor: it reports the same selection
// to the input system again, in order, overriding the editor's report. The
// editor's own selection, and so the end that Shift+arrow moves, is kept.
func (a *App) orderIMESelection(gtx layout.Context) {
	caret, anchor := a.editor.Selection()
	if caret <= anchor {
		return
	}
	size := float32(gtx.Sp(editorTextSize))
	gtx.Execute(key.SelectionCmd{
		Tag:   &a.editor,
		Range: key.Range{Start: anchor, End: caret},
		// Where the browser puts its hidden text field, for input method
		// windows; the font's ascent and descent are about 0.9 and 0.22 em.
		Caret: key.Caret{Pos: a.editor.CaretCoords(), Ascent: 0.9 * size, Descent: 0.22 * size},
	})
}
