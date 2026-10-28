//go:build !js

package ui

import (
	"image"
	"testing"
)

// rightClickMiss right-clicks the i-th underlined word in the editor.
func (h *harness) rightClickMiss(i int) {
	h.t.Helper()
	if len(h.a.spell.boxes) <= i {
		h.t.Fatalf("only %d words underlined", len(h.a.spell.boxes))
	}
	r := h.a.spell.boxes[i].r
	// The editor's origin: below the bars, inside its 14×12 dp inset.
	at := r.Min.Add(r.Max).Div(2).Add(image.Pt(28, h.a.barH+h.a.toolH+24))
	h.rightClick(float32(at.X), float32(at.Y))
}

func TestSpellingCorrection(t *testing.T) {
	h := newHarness(t, "")
	h.setText("Hello wrold, and `wrold` in code.\n", 0, 0)
	if n := len(h.a.spell.misses); n != 1 {
		t.Fatalf("%d misspellings, want 1 (code isn't checked)", n)
	}
	h.rightClickMiss(0)
	items := h.a.ctx.spellItems
	if !h.a.ctx.open || len(items) == 0 || items[0].label != "world" {
		t.Fatalf("context menu should offer world first, got %v", labels(items))
	}
	h.run(items[0])
	h.expect("Hello world, and `wrold` in code.\n")
	if len(h.a.spell.misses) != 0 {
		t.Error("the correction is still underlined")
	}
}

func TestSpellingAddToDictionary(t *testing.T) {
	h := newHarness(t, "")
	h.setText("A word called Doetjesz.\n", 0, 0)
	h.rightClickMiss(0)
	items := h.a.ctx.spellItems
	add := items[len(items)-3]
	if add.label != "Add “Doetjesz” to Dictionary" {
		t.Fatalf("expected Add to Dictionary, got %v", labels(items))
	}
	h.run(add)
	if len(h.a.spell.misses) != 0 {
		t.Error("the added word is still misspelt")
	}
	if w := loadSettings().Words; len(w) != 1 || w[0] != "Doetjesz" {
		t.Errorf("saved words %q", w)
	}
}

// A right-click elsewhere offers no corrections, and the word being typed
// isn't underlined until the caret leaves it.
func TestSpellingNoMenuAway(t *testing.T) {
	h := newHarness(t, "")
	h.setText("Some wrold\n", 10, 10)
	if len(h.a.spell.boxes) != 0 {
		t.Error("the word at the caret is underlined while typing")
	}
	h.rightClick(600, 700)
	if !h.a.ctx.open || h.a.ctx.spellItems != nil {
		t.Error("a right-click away from a misspelling offers corrections")
	}
}

func TestSpellingToggle(t *testing.T) {
	h := newHarness(t, "")
	h.setText("Some wrold\n", 0, 0)
	h.run(h.a.cmds.spelling)
	if len(h.a.spell.misses) != 0 || h.a.cmds.spelling.label != "Show Spelling Errors" {
		t.Error("hiding spelling errors did not work")
	}
	h.run(h.a.cmds.spelling)
	if len(h.a.spell.misses) != 1 {
		t.Error("showing spelling errors did not work")
	}
}

func labels(cs []*command) []string {
	var out []string
	for _, c := range cs {
		if c != nil {
			out = append(out, c.label)
		}
	}
	return out
}
