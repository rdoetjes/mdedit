package ui

import (
	"strings"
	"testing"

	"gioui.org/io/key"
)

func TestEnterContinuesLists(t *testing.T) {
	h := newHarness(t, "")
	if !h.router.Source().Focused(&h.a.editor) {
		t.Fatal("editor not focused")
	}
	h.setText("1. one", 6, 6)
	h.press(key.NameReturn, 0)
	h.expect("1. one\n2. ")
	h.press(key.NameReturn, 0)
	h.expect("1. one\n\n")

	h.setText("- a", 3, 3)
	h.press(key.NameReturn, 0)
	h.expect("- a\n- ")

	h.setText("plain", 5, 5)
	h.press(key.NameReturn, 0)
	h.expect("plain\n")
}

func TestShortcuts(t *testing.T) {
	h := newHarness(t, "")
	h.setText("word", 0, 4)
	h.press("B", key.ModShortcut)
	h.expect("**word**")
	h.press("B", key.ModShortcut)
	h.expect("word")
	h.press("1", key.ModShortcut)
	h.expect("# word")

	// On an empty line the marker is inserted, ready for typing.
	h.setText("", 0, 0)
	h.press("2", key.ModShortcut)
	h.expect("## ")
	h.expectCaret(3)
}

func TestTabKey(t *testing.T) {
	h := newHarness(t, "")
	h.setText("- a\n- b", 7, 7)
	h.press(key.NameTab, 0)
	h.expect("- a\n    - b")
	if !h.router.Source().Focused(&h.a.editor) {
		t.Error("Tab moved focus out of the editor")
	}
	h.press(key.NameTab, key.ModShift)
	h.expect("- a\n- b")
	h.setText("x", 1, 1)
	h.a.editor.Insert("\tpasted")
	h.frame()
	h.frame()
	h.expect("x   pasted")
}

func TestMacNavigation(t *testing.T) {
	if key.ModShortcut != key.ModCommand {
		t.Skip("macOS key bindings")
	}
	h := newHarness(t, "")
	cmd := key.ModCommand
	h.setText("abc\ndefgh", 6, 6)
	h.press(key.NameLeftArrow, cmd)
	h.expectCaret(4)
	h.press(key.NameRightArrow, cmd)
	h.expectCaret(9)
	h.press(key.NameUpArrow, cmd)
	h.expectCaret(0)
	h.press(key.NameDownArrow, cmd|key.ModShift)
	h.expectSelection(9, 0)
	h.setText("abc\ndefgh", 6, 6)
	h.press(key.NameDeleteBackward, cmd)
	h.expect("abc\nfgh")
}

// A long line wraps; ⌘← and ⌘→ stay on the screen line.
func TestMacNavigationWrapped(t *testing.T) {
	if key.ModShortcut != key.ModCommand {
		t.Skip("macOS key bindings")
	}
	h := newHarness(t, "")
	long := strings.Repeat("word ", 400)
	h.setText(long, 1000, 1000)
	h.press(key.NameLeftArrow, key.ModCommand)
	start, _ := h.a.editor.Selection()
	if start <= 0 || start >= 1000 {
		t.Errorf("⌘← in wrapped line went to %d, want a screen line start in (0, 1000)", start)
	}
	h.press(key.NameRightArrow, key.ModCommand)
	end, _ := h.a.editor.Selection()
	if end <= 1000 || end >= len(long) {
		t.Errorf("⌘→ in wrapped line went to %d, want a screen line end in (1000, %d)", end, len(long))
	}
}
