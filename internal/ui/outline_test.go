//go:build !js

package ui

import (
	"strings"
	"testing"
)

const outlineSample = "# Guide\n\nIntro.\n\n## Install\n\nSteps.\n\n### Linux\n\nText.\n\n## Usage\n\nMore.\n"

func TestOutlineItems(t *testing.T) {
	h := newHarness(t, "")
	h.setText(outlineSample+"```\n# not a heading\n```\n", 0, 0)
	var got []string
	for _, it := range h.a.outline.items {
		got = append(got, strings.Repeat("#", it.level)+" "+it.text)
	}
	want := "# Guide|## Install|### Linux|## Usage"
	if strings.Join(got, "|") != want {
		t.Errorf("outline %q, want %q", got, want)
	}
	if it := h.a.outline.items[2]; string([]rune(outlineSample)[it.offset:it.offset+9]) != "### Linux" {
		t.Errorf("Linux item at offset %d", it.offset)
	}
}

// The heading of the section holding the caret is highlighted.
func TestOutlineFollowsCaret(t *testing.T) {
	h := newHarness(t, "")
	h.showOutline()
	h.setText(outlineSample, strings.Index(outlineSample, "Text."), 0)
	if h.a.outline.current != 2 {
		t.Errorf("current item %d, want 2 (Linux)", h.a.outline.current)
	}
}

// Clicking a heading far down puts it at the top of the editor.
func TestOutlineClickJumps(t *testing.T) {
	h := newHarness(t, "")
	h.showOutline()
	filler := strings.Repeat("Filler line.\n", 300)
	text := "# Top\n\n" + filler + "## Far down\n\n" + filler
	h.setText(text, 0, 0)
	// The outline's second row: below the bars and the OUTLINE header.
	h.click(100, float32(h.a.barH+h.a.toolH+150))
	want := strings.Index(text, "## Far down")
	h.expectCaret(want)
	if y := h.a.editor.CaretCoords().Y; y < 0 || y > 100 {
		t.Errorf("the heading is at y=%v in the editor, want it at the top", y)
	}
	if !h.a.outline.open || h.a.outline.current != 1 {
		t.Errorf("current item %d, want 1", h.a.outline.current)
	}
}

// showOutline opens the sidebar; the setting persists between tests.
func (h *harness) showOutline() {
	if !h.a.outline.open {
		h.run(h.a.cmds.outline)
	}
}
