//go:build !js

package ui

import (
	"strings"
	"testing"
)

// Right-clicking keeps the selection, so Cut and Copy apply to it.
func TestRightClickKeepsSelection(t *testing.T) {
	h := newHarness(t, "")
	h.setText("# A\n\n## B\n", 0, 0)
	h.a.editor.SetCaret(4, 0)
	h.rightClick(600, 700) // in the editor pane, away from the selection
	h.expectSelection(4, 0)
	if !h.a.ctx.open {
		t.Error("right-click did not open the context menu")
	}
}

// Right-clicking after scrolling away from the caret doesn't scroll back.
func TestRightClickKeepsScroll(t *testing.T) {
	h := newHarness(t, "")
	h.setText(strings.Repeat("Some line of text.\n", 400), 0, 0)
	h.scroll(600, 700, 3000)
	before := h.a.editor.CaretCoords()
	if before.Y > -1000 {
		t.Fatalf("the editor did not scroll: caret at %v", before)
	}
	h.rightClick(600, 700)
	if after := h.a.editor.CaretCoords(); after != before {
		t.Errorf("right-click scrolled the editor: caret moved from %v to %v on screen", before, after)
	}
	h.expectSelection(0, 0)
}

func TestTOCContextMenu(t *testing.T) {
	h := newHarness(t, "")
	h.setText("# A\n\n## B\n", 0, 0)
	h.rightClick(600, 700)
	if last := h.lastContextItem(); last != h.a.cmds.insertTOC {
		t.Fatalf("expected Insert Table of Contents, got %q", last.label)
	}
	h.run(h.a.cmds.insertTOC)
	// Inserted on its own lines below the line holding the caret: the title.
	h.expect("# A\n\n<!-- toc -->\n\n- [A](#a)\n  - [B](#b)\n\n<!-- /toc -->\n\n## B\n")

	// Add a heading, then update from the context menu (in the preview pane).
	h.setText(h.a.editor.Text()+"\n## C\n", 0, 0)
	h.rightClick(1800, 500)
	if !h.a.ctx.open || h.lastContextItem() != h.a.cmds.updateTOC {
		t.Fatal("expected Update Table of Contents in the context menu")
	}
	h.run(h.a.cmds.updateTOC)
	h.expect("# A\n\n<!-- toc -->\n\n- [A](#a)\n  - [B](#b)\n  - [C](#c)\n\n<!-- /toc -->\n\n## B\n\n## C\n")
	h.expectAnchors("a", "b", "c")
}

func (h *harness) lastContextItem() *command {
	items := h.a.contextItems()
	return items[len(items)-1]
}

// expectAnchors checks that heading anchors resolve in the preview.
func (h *harness) expectAnchors(ids ...string) {
	h.t.Helper()
	for _, id := range ids {
		if h.a.doc.HeadingIndex(id) < 0 {
			h.t.Errorf("anchor #%s not found", id)
		}
	}
}
