//go:build !js

package ui

import (
	"testing"

	"gioui.org/io/key"
	"gioui.org/layout"

	"mdedit/internal/pdfout"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		in   string
		want float32
		ok   bool
	}{
		{"12", 12, true},
		{"10.5", 10.5, true},
		{"10,5", 10.5, true}, // a Dutch keyboard
		{"0", 1, true},       // clamped
		{"0.2", 1, true},
		{"500", 80, true},
		{"", 0, false},
		{".", 0, false},
		{"1.2.3", 0, false},
	}
	for _, tt := range tests {
		if got, ok := parseSize(tt.in); got != tt.want || ok != tt.ok {
			t.Errorf("parseSize(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestStepSize(t *testing.T) {
	tests := []struct {
		v    float32
		dir  int
		want float32
	}{
		{10.5, 1, 11}, // to the next whole size
		{10.5, -1, 10},
		{12, 1, 13},
		{12, -1, 11},
		{1, -1, 1}, // the limits
		{80, 1, 80},
		{79.5, 1, 80},
	}
	for _, tt := range tests {
		if got := stepSize(tt.v, tt.dir); got != tt.want {
			t.Errorf("stepSize(%v, %d) = %v, want %v", tt.v, tt.dir, got, tt.want)
		}
	}
}

// The preview follows the size in proportion, is unchanged at the default,
// and the size is remembered.
func TestTextSize(t *testing.T) {
	h := newHarness(t, "")
	h.setText("Some text.\n", 0, 0)
	if h.a.textSize != pdfout.DefaultFontSize || h.a.preview.ctx.Em != 32 || h.a.sizeBox.field.Text() != "10.5" {
		t.Fatalf("default: size %v, em %v, field %q", h.a.textSize, h.a.preview.ctx.Em, h.a.sizeBox.field.Text())
	}
	before := h.a.preview.blockLayout(0).Height
	h.a.sizeBox.set(21)
	h.frame()
	if h.a.preview.ctx.Em != 64 {
		t.Errorf("em %v at double the size, want 64", h.a.preview.ctx.Em)
	}
	if after := h.a.preview.blockLayout(0).Height; after <= before {
		t.Errorf("block height %v, was %v: the preview wasn't laid out again", after, before)
	}
	if next := h.restart(); next.a.textSize != 21 || next.a.sizeBox.field.Text() != "21" {
		t.Errorf("after a restart: size %v, field %q", next.a.textSize, next.a.sizeBox.field.Text())
	}
}

// Extreme sizes must draw, with text, without trouble.
func TestTextSizeLimits(t *testing.T) {
	h := newHarness(t, "")
	h.setText("# Title\n\nSome **text** and $x^2$ with `code`.\n\n| a | b |\n|---|---|\n| 1 | 2 |\n", 0, 0)
	for _, v := range []float32{minTextSize, maxTextSize} {
		h.a.sizeBox.set(v)
		h.frame()
		h.frame()
	}
}

func (h *harness) focusSize() *sizeBox {
	b := &h.a.sizeBox
	h.a.focusEditor = false
	h.run(newCommand("", "", 0, h.focusSizeCmd))
	return b
}

// A typed size counts on Enter, not at each digit, and the editor gets the
// keyboard back.
func TestTypingASize(t *testing.T) {
	h := newHarness(t, "")
	b := h.focusSize()
	b.field.SetText("1")
	h.frame()
	if h.a.textSize != pdfout.DefaultFontSize {
		t.Errorf("size %v before Enter", h.a.textSize)
	}
	b.field.SetText("14")
	h.press(key.NameReturn, 0)
	h.frame()
	if h.a.textSize != 14 || !h.editorFocused() {
		t.Errorf("size %v, editor focused %v", h.a.textSize, h.editorFocused())
	}
}

func TestSizeKeys(t *testing.T) {
	h := newHarness(t, "")
	b := h.focusSize()
	h.press(key.NameUpArrow, 0)
	if h.a.textSize != 11 {
		t.Errorf("size %v after Up, want 11", h.a.textSize)
	}
	h.press(key.NameDownArrow, 0)
	h.press(key.NameDownArrow, 0)
	if h.a.textSize != 9 {
		t.Errorf("size %v after Down twice, want 9", h.a.textSize)
	}
	b.field.SetText("33")
	h.frame()
	h.press(key.NameEscape, 0)
	if b.field.Text() != "9" || h.a.textSize != 9 {
		t.Errorf("Escape: field %q, size %v", b.field.Text(), h.a.textSize)
	}
}

// Rubbish in the field goes back to the current size when leaving it.
func TestBadSizeRestored(t *testing.T) {
	h := newHarness(t, "")
	b := h.focusSize()
	b.field.SetText(".")
	h.frame()
	h.a.focusEditor = true
	h.frame()
	h.frame()
	if b.field.Text() != "10.5" || h.a.textSize != pdfout.DefaultFontSize {
		t.Errorf("field %q, size %v", b.field.Text(), h.a.textSize)
	}
}

func (h *harness) focusSizeCmd(gtx layout.Context) {
	gtx.Execute(key.FocusCmd{Tag: &h.a.sizeBox.field})
}

// The arrows work with real clicks. The box comes after the font box: its
// 12 dp gap, the 36 dp field, and the middle of the 18 dp arrows.
func TestSizeArrowsClick(t *testing.T) {
	h := newHarness(t, "")
	f, b := &h.a.fontBox, &h.a.sizeBox
	x := float32(f.x + f.box.Max.X + 24 + 72 + 2 + 18)
	top := float32(h.a.barH + 6)
	h.click(x, top+float32(b.h)/4)
	if h.a.textSize != 11 {
		t.Errorf("size %v after clicking up, want 11", h.a.textSize)
	}
	h.click(x, top+float32(b.h)*3/4)
	h.click(x, top+float32(b.h)*3/4)
	if h.a.textSize != 9 {
		t.Errorf("size %v after clicking down twice, want 9", h.a.textSize)
	}
}
