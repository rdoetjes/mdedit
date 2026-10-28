//go:build !js

package ui

import (
	"testing"

	"gioui.org/io/key"
)

const findSample = "one two one three One"

func TestFindNavigation(t *testing.T) {
	h := newHarness(t, "")
	h.setText(findSample, 0, 0)
	h.press("F", key.ModShortcut)
	if !h.a.find.open || !h.router.Source().Focused(&h.a.find.query) {
		t.Fatal("Ctrl/⌘+F did not open the find bar with the search field focused")
	}
	h.a.find.query.SetText("one")
	h.frame()
	h.expectCounter("3 matches")
	h.press(key.NameReturn, 0)
	h.expectSelection(3, 0)
	h.expectCounter("1 of 3")
	h.press(key.NameReturn, 0)
	h.expectSelection(11, 8)
	h.press(key.NameReturn, 0)
	h.expectSelection(21, 18) // "One": case-insensitive
	h.press(key.NameReturn, 0)
	h.expectSelection(3, 0) // wraps around
	h.press(key.NameReturn, key.ModShift)
	h.expectSelection(21, 18) // Shift+Enter goes back, wrapping
	h.press(key.NameEscape, 0)
	if h.a.find.open || !h.router.Source().Focused(&h.a.editor) {
		t.Error("Escape did not close the bar and return to the editor")
	}
}

func TestFindMatchCase(t *testing.T) {
	h := newHarness(t, "")
	h.setText(findSample, 0, 0)
	h.press("F", key.ModShortcut)
	h.a.find.query.SetText("One")
	h.a.find.matchCase.Value = true
	h.frame()
	h.expectCounter("1 match")
	h.a.find.query.SetText("xyz")
	h.frame()
	h.expectCounter("No results")
}

func TestReplace(t *testing.T) {
	h := newHarness(t, "")
	h.setText(findSample, 0, 0)
	h.press("R", key.ModShortcut)
	if !h.a.find.open || !h.a.find.replace {
		t.Fatal("Ctrl/⌘+R did not open the replace bar")
	}
	h.a.find.query.SetText("one")
	h.a.find.repl.SetText("1")
	h.frame()
	h.a.find.replOne.Click() // nothing selected yet: selects the first match
	h.frame()
	h.expectSelection(3, 0)
	h.a.find.replOne.Click() // replaces it and moves on
	h.frame()
	h.expect("1 two one three One")
	h.expectSelection(9, 6)
	h.a.find.replAll.Click()
	h.frame()
	h.expect("1 two 1 three 1")
	h.press(key.NameEscape, 0)    // back in the editor…
	h.press("Z", key.ModShortcut) // …one undo step undoes Replace All
	h.expect("1 two one three One")
}

// The selection becomes the search text.
func TestFindUsesSelection(t *testing.T) {
	h := newHarness(t, "")
	h.setText(findSample, 4, 7)
	h.press("F", key.ModShortcut)
	if got := h.a.find.query.Text(); got != "two" {
		t.Errorf("search text %q, want %q", got, "two")
	}
}

func (h *harness) expectCounter(want string) {
	h.t.Helper()
	h.a.find.update()
	if got := h.a.find.counter(); got != want {
		h.t.Errorf("counter %q, want %q", got, want)
	}
}
