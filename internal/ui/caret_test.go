//go:build !js

package ui

import (
	"strings"
	"testing"
)

func TestCaretChar(t *testing.T) {
	text := []rune("ab\nç")
	tests := map[int]string{0: "a", 1: "b", 2: " ", 3: "ç", 4: " ", -1: " "}
	for at, want := range tests {
		if got := caretChar(text, at); got != want {
			t.Errorf("caretChar at %d = %q, want %q", at, got, want)
		}
	}
}

// The editor draws the block caret instead of Gio's line; drawing it at
// line ends, the end of the text, in wrapped lines or with a selection
// must work.
func TestBlockCaret(t *testing.T) {
	h := newHarness(t, "")
	if !h.a.editor.HideCaret {
		t.Fatal("Gio's own caret is still drawn")
	}
	long := strings.Repeat("wrapped words ", 300)
	text := "# Title\n\n" + long + "\n\nlast ç\n"
	for _, at := range []int{0, 7, 8, 9 + len(long)/2, len([]rune(text)) - 2, len([]rune(text))} {
		h.setText(text, at, at)
		h.frame() // draws the caret; must not panic at line ends or the end
	}
	h.setText("", 0, 0)
	h.frame()
	h.setText("selected text", 0, 8)
	h.frame()
	if !h.editorFocused() {
		t.Error("the editor should have the focus, so the caret shows")
	}
}
