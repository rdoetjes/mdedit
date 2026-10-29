//go:build !js

package ui

import (
	"testing"
	"unicode/utf8"

	"gioui.org/io/key"
)

// browserTypes types text the way Gio's browser port does (the textarea
// "input" handler in gioui.org/app/os_js.go, case "insertText"): it
// replaces the IME selection and puts the caret at Range.Start plus the
// length of the text.
func (h *harness) browserTypes(text string) {
	r := h.router.EditorState().Selection.Range
	caret := r.Start + utf8.RuneCountInString(text)
	h.router.Queue(key.EditEvent{Range: r, Text: text}, key.SelectionEvent{Start: caret, End: caret})
	h.frame()
	h.frame()
}

// Typing over a word selected left to right (a double-click or a drag)
// replaces it and continues right after the typed text.
func TestBrowserTypingOverSelection(t *testing.T) {
	reorderIMESelection = true
	defer stopReordering()
	h := newHarness(t, "")
	h.setText("say hello world", 9, 4) // "hello", caret at its right end
	h.browserTypes("b")
	h.browserTypes("y")
	h.browserTypes("e")
	h.expect("say bye world")
	h.expectCaret(7)
}

func stopReordering() { reorderIMESelection = false }
