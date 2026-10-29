//go:build !js

package ui

import (
	"strings"
	"testing"

	"mdedit/internal/mdmodel"
	"mdedit/internal/render"
)

func TestWordAt(t *testing.T) {
	tests := []struct {
		s        string
		i        int
		from, to int
		ok       bool
	}{
		{"the quick fox", 5, 4, 9, true}, // inside "quick"
		{"the quick fox", 3, 4, 9, true}, // on a space: the next word
		{"the fox.", 8, 4, 7, true},      // after the end: the last word
		{"— 3.", 0, 2, 3, true},          // a number
		{" — ", 1, 0, 0, false},          // no word at all
		{"café au lait", 2, 0, 4, true},  // non-ASCII letters
	}
	for _, tt := range tests {
		from, to, ok := wordAt([]rune(tt.s), tt.i)
		if from != tt.from || to != tt.to || ok != tt.ok {
			t.Errorf("wordAt(%q, %d) = %d, %d, %v; want %d, %d, %v", tt.s, tt.i, from, to, ok, tt.from, tt.to, tt.ok)
		}
	}
}

const sourceSample = "# Foxes\n\nThe **quick** fox and the [other\nfox](https://example.com) jump.\n\n$$\nx^2\n$$\n"

func TestSourceOffset(t *testing.T) {
	text := []rune(sourceSample)
	doc := mdmodel.Parse([]byte(sourceSample))
	second := strings.Index(sourceSample, "fox](")
	tests := []struct {
		name string
		c    SourceClick
		want int
	}{
		{"first fox", SourceClick{Block: 1, Word: "fox", Nth: 0, Offset: 1}, strings.Index(sourceSample, "fox and") + 1},
		{"second fox, in a link, after a line break", SourceClick{Block: 1, Word: "fox", Nth: 1}, second},
		{"bold word", SourceClick{Block: 1, Word: "quick", Offset: 5}, strings.Index(sourceSample, "quick") + 5},
		{"heading", SourceClick{Block: 0, Word: "Foxes", Offset: 2}, 4},
		{"maths: the formula", SourceClick{Block: 2}, strings.Index(sourceSample, "x^2")},
		{"word not in source", SourceClick{Block: 1, Word: "zebra"}, strings.Index(sourceSample, "The")},
		{"more occurrences than in the source", SourceClick{Block: 1, Word: "fox", Nth: 9}, second},
	}
	for _, tt := range tests {
		if got := sourceOffset(text, doc, tt.c); got != tt.want {
			t.Errorf("%s: offset %d, want %d", tt.name, got, tt.want)
		}
	}
}

// previewPoint returns the window coordinates of character n of the k-th
// text primitive of block b in the preview, which must be scrolled to the
// top, in the split view.
func (h *harness) previewPoint(b, k, n int) (float32, float32) {
	h.t.Helper()
	p := h.a.preview
	y := float32(p.pad)
	for i := range b {
		y += p.blockLayout(i).Height
	}
	pr := textPrims(p.blockLayout(b))[k]
	left := float32(int(h.a.split*float32(h.size.X)) + 12) // the editor and the divider (6 dp)
	x := pr.X + p.text.measure(pr.Font, pr.Size, string([]rune(pr.Text)[:n])).Width + 2
	return left + p.ox + x, float32(h.a.barH+h.a.toolH) + y + pr.Y - pr.Size/3
}

func textPrims(l *render.Layout) []render.Prim {
	var out []render.Prim
	for _, pr := range l.Prims {
		if pr.Kind == render.PText {
			out = append(out, pr)
		}
	}
	return out
}

// Clicking a word in the preview puts the caret on it in the editor.
func TestPreviewClickShowsSource(t *testing.T) {
	h := newHarness(t, "")
	h.setText(sourceSample, 0, 0)
	h.a.preview.ScrollToBlock(0)
	h.frame()
	prims := textPrims(h.a.preview.blockLayout(1))
	k := -1
	for i, pr := range prims {
		if strings.Contains(pr.Text, "fox and") {
			k = i
		}
	}
	if k < 0 {
		t.Fatalf("no text with \"fox and\" in %v", prims)
	}
	n := strings.Index(prims[k].Text, "fox")
	h.click(h.previewPoint(1, k, n+1))
	h.expectCaret(strings.Index(sourceSample, "fox and") + 1)
}

// A click on a link still follows the link instead of moving the caret.
func TestPreviewLinkClickFollowsLink(t *testing.T) {
	h := newHarness(t, "")
	h.setText("See [the heading](#foxes) here.\n\n# Foxes\n", 0, 0)
	h.a.preview.ScrollToBlock(0)
	h.frame()
	var followed string
	h.a.preview.OnLink = recordLink{&followed}.follow
	prims := textPrims(h.a.preview.blockLayout(0))
	for k, pr := range prims {
		if strings.HasPrefix(pr.Text, "the heading") {
			h.click(h.previewPoint(0, k, 2))
		}
	}
	if followed != "#foxes" {
		t.Errorf("followed %q, want #foxes", followed)
	}
	h.expectCaret(0)
}

type recordLink struct{ url *string }

func (r recordLink) follow(url string) { *r.url = url }

// wrappedSample has paragraphs that wrap over many lines in the editor.
var wrappedSample = "# Top\n\n" + strings.Repeat("Some words that wrap. ", 200) + "\n\n## Middle\n\n" +
	strings.Repeat("More words here. ", 200) + "\n\nThe fox is near the end.\n"

// The caret's line counts lines of the text, not wrapped lines on screen.
func TestCaretLineIgnoresWrapping(t *testing.T) {
	h := newHarness(t, "")
	at := strings.Index(wrappedSample, "fox")
	h.setText(wrappedSample, at, at)
	if line, col := h.a.caretLine(); line != 8 || col != 4 {
		t.Errorf("caret at line %d, column %d; want 8, 4", line, col)
	}
}

// After a click in the preview the clicked word is on the editor's top
// line, the preview stays where it is, and the editor has the keyboard
// focus, even if the find bar had it.
func TestPreviewClickKeepsPreviewAndFocusesEditor(t *testing.T) {
	h := newHarness(t, "")
	h.setText(wrappedSample, 0, 0)
	h.run(h.a.cmds.find)
	if !h.a.find.open || h.editorFocused() {
		t.Fatal("the find bar should have the focus")
	}
	const middle = 2 // the heading after the first wrapped paragraph
	h.a.preview.ScrollToBlock(middle)
	h.frame()
	before := h.a.preview.list.Position
	h.click(h.previewPointAt(middle, 0, 1))
	h.expectCaret(strings.Index(wrappedSample, "Middle") + 1)
	if y := h.a.editor.CaretCoords().Y; y < 0 || y > 60 {
		t.Errorf("the clicked word is at y=%v in the editor, want it on the top line", y)
	}
	if after := h.a.preview.list.Position; after.First != before.First || after.Offset != before.Offset {
		t.Errorf("the preview scrolled from %+v to %+v", before, after)
	}
	if !h.editorFocused() {
		t.Error("the editor did not get the keyboard focus back")
	}
}

// previewPointAt is previewPoint for a block scrolled to the top of the
// preview.
func (h *harness) previewPointAt(b, k, n int) (float32, float32) {
	p := h.a.preview
	pr := textPrims(p.blockLayout(b))[k]
	left := float32(int(h.a.split*float32(h.size.X)) + 12)
	x := pr.X + p.text.measure(pr.Font, pr.Size, string([]rune(pr.Text)[:n])).Width + 2
	return left + p.ox + x, float32(h.a.barH+h.a.toolH) + pr.Y - pr.Size/3
}

func (h *harness) editorFocused() bool { return h.router.Source().Focused(&h.a.editor) }
