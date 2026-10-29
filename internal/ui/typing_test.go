//go:build !js

package ui

import (
	"testing"
	"time"

	"gioui.org/io/key"
)

func TestChangedRange(t *testing.T) {
	tests := []struct {
		old, new           string
		from, oldTo, newTo int
	}{
		{"hello world", "hello, world", 5, 5, 6},
		{"hello world", "hello", 5, 11, 5},
		{"abc", "abc", 3, 3, 3},
		{"", "x", 0, 0, 1},
		{"café au lait", "cafè au lait", 3, 4, 4}, // é and è share a lead byte
		{"aaaa", "aaaaa", 4, 4, 5},
		{"ëë", "ë", 1, 2, 1},
	}
	for _, tt := range tests {
		from, oldTo, newTo := changedRange(tt.old, tt.new)
		if from != tt.from || oldTo != tt.oldTo || newTo != tt.newTo {
			t.Errorf("changedRange(%q, %q) = %d, %d, %d, want %d, %d, %d",
				tt.old, tt.new, from, oldTo, newTo, tt.from, tt.oldTo, tt.newTo)
		}
		o, n := []rune(tt.old), []rune(tt.new)
		if string(o[:from]) != string(n[:from]) || string(o[oldTo:]) != string(n[newTo:]) {
			t.Errorf("changedRange(%q, %q): the parts outside the change differ", tt.old, tt.new)
		}
	}
}

// In a document that is slow to parse, typing postpones the reparse and
// moves the wavy lines along; the reparse follows when typing pauses.
func TestTypingPostponesReparse(t *testing.T) {
	h := newHarness(t, "")
	text := "Some wrold here. Another wrold end.\n"
	h.setText(text, 0, 0)
	h.a.reparseCost = time.Second // pretend the document is huge
	version := h.a.version
	h.router.Queue(key.EditEvent{Range: key.Range{Start: 0, End: 0}, Text: "Ok. "})
	h.frame()
	if h.a.reparseDue.IsZero() || h.a.version != version+1 {
		t.Fatal("typing did not postpone the reparse")
	}
	got := h.a.editor.Text()
	for _, m := range h.a.spell.misses {
		if w := string([]rune(got)[m.Start:m.End]); w != "wrold" {
			t.Errorf("a wavy line is under %q after typing, want wrold", w)
		}
	}
	if len(h.a.spell.misses) != 2 {
		t.Errorf("%d misses after typing before them, want 2", len(h.a.spell.misses))
	}
	h.a.reparseDue = time.Now().Add(-time.Millisecond) // the pause is over
	h.frame()
	if !h.a.reparseDue.IsZero() || h.a.parsedText != got {
		t.Error("the postponed reparse did not run")
	}
}

// Typing inside a misspelt word removes its wavy line until the reparse.
func TestTypingInsideMissDropsIt(t *testing.T) {
	h := newHarness(t, "")
	h.setText("Some wrold here and wrold there.\n", 0, 0)
	h.a.reparseCost = time.Second
	h.router.Queue(key.EditEvent{Range: key.Range{Start: 7, End: 7}, Text: "x"})
	h.frame()
	if len(h.a.spell.misses) != 1 || h.a.spell.misses[0].Start != 21 {
		t.Errorf("misses %v, want only the second wrold, moved to 21", h.a.spell.misses)
	}
}
