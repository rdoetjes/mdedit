//go:build !js

package ui

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"golang.org/x/image/font/gofont/gomono"

	"mdedit/internal/render"
	"mdedit/internal/sysfont"
)

// monoFamily installs a copy of Go Mono as a "system font". Monospaced
// text shows the font is used: all letters are equally wide.
func monoFamily(t *testing.T) sysfont.Family {
	p := filepath.Join(t.TempDir(), "test.ttf")
	if err := os.WriteFile(p, gomono.TTF, 0o644); err != nil {
		t.Fatal(err)
	}
	f := sysfont.Family{Name: "Test Mono"}
	f.Faces[sysfont.Regular] = &sysfont.Face{Path: p}
	return f
}

func (h *harness) bodyIsMono() bool {
	m := h.a.preview.text
	return m.measure(render.FontSpec{}, 20, "iiii").Width == m.measure(render.FontSpec{}, 20, "MMMM").Width
}

func (h *harness) chooseFont(name string, fams ...sysfont.Family) {
	h.a.families = fams
	h.a.fontBox.setFamilies(fams)
	h.a.fontBox.choose(name)
	h.frame()
	h.frame()
}

func TestDocumentFont(t *testing.T) {
	h := newHarness(t, "")
	h.setText("Text in the font.\n", 0, 0)
	f := monoFamily(t)
	if h.bodyIsMono() {
		t.Fatal("the built-in text font should not be monospaced")
	}
	h.chooseFont("Test Mono", f)
	if !h.bodyIsMono() || h.a.bodyFont.Name != "Test Mono" {
		t.Fatal("the chosen font is not used in the preview")
	}
	if !h.editorFocused() {
		t.Error("the editor should get the keyboard focus back")
	}
	next := h.restart()
	if !next.bodyIsMono() || next.a.fontBox.field.Text() != "Test Mono" {
		t.Errorf("after a restart the box shows %q", next.a.fontBox.field.Text())
	}
	h.chooseFont(defaultFontName, f)
	if h.bodyIsMono() || h.a.bodyFont.Name != "" || loadSettings().Font != nil {
		t.Error("going back to the built-in font failed")
	}
}

// A saved font whose files are gone leaves the built-in font, with a
// message, and doesn't stop mdedit from starting.
func TestDocumentFontGone(t *testing.T) {
	h := newHarness(t, "")
	f := monoFamily(t)
	h.chooseFont("Test Mono", f)
	os.Remove(f.Faces[sysfont.Regular].Path)
	next := h.restart()
	if next.bodyIsMono() || !strings.Contains(next.a.status, "Test Mono") {
		t.Errorf("status %q; want the built-in font and a message", next.a.status)
	}
}

// comboPoint returns the window position of a point in the font box.
func (h *harness) comboPoint(p image.Point) (float32, float32) {
	return float32(h.a.fontBox.x + p.X), float32(h.a.barH + 6 + p.Y) // the toolbar's 3 dp margin
}

// openCombo clicks the field, which drops the list.
func (h *harness) openCombo(fams ...sysfont.Family) *fontCombo {
	c := &h.a.fontBox
	h.a.families = fams
	c.setFamilies(fams)
	h.frame()
	h.click(h.comboPoint(c.box.Min.Add(image.Pt(30, 10))))
	h.frame()
	if !c.open {
		h.t.Fatal("clicking the field did not drop the list")
	}
	return c
}

func (h *harness) typeInCombo(s string) {
	h.a.fontBox.field.SetText(s)
	h.frame()
	h.frame()
}

func TestComboFilters(t *testing.T) {
	h := newHarness(t, "")
	c := h.openCombo(sysfont.Family{Name: "Arial"}, sysfont.Family{Name: "Georgia"}, sysfont.Family{Name: "Arial Black"})
	if len(c.shown) != 4 {
		t.Errorf("shown %q; the chosen name in the field should not filter", c.shown)
	}
	h.typeInCombo("ARIAL")
	if !slices.Equal(c.shown, []string{"Arial", "Arial Black"}) {
		t.Errorf("shown %q", c.shown)
	}
	h.press(key.NameEscape, 0)
	if c.open || c.field.Text() != defaultFontName || h.a.bodyFont.Name != "" {
		t.Errorf("Escape: open %v, field %q", c.open, c.field.Text())
	}
}

// Clicking an entry chooses the font; the field shows it.
func TestComboClickChooses(t *testing.T) {
	h := newHarness(t, "")
	h.setText("Text.\n", 0, 0)
	c := h.openCombo(monoFamily(t))
	h.typeInCombo("mono")
	if len(c.shown) != 1 {
		t.Fatalf("shown %q", c.shown)
	}
	h.click(h.comboPoint(c.dropped.Min.Add(image.Pt(20, 12))))
	if c.open || c.field.Text() != "Test Mono" || !h.bodyIsMono() {
		t.Errorf("open %v, field %q, font used %v", c.open, c.field.Text(), h.bodyIsMono())
	}
	if !h.editorFocused() {
		t.Error("the editor should get the keyboard focus back")
	}
}

// Enter chooses the highlighted entry, and the arrow keys move it.
func TestComboKeyboard(t *testing.T) {
	h := newHarness(t, "")
	c := h.openCombo(monoFamily(t))
	h.press(key.NameDownArrow, 0)
	if c.hot != 1 {
		t.Fatalf("highlighted entry %d, want 1", c.hot)
	}
	h.press(key.NameReturn, 0)
	if h.a.bodyFont.Name != "Test Mono" {
		t.Errorf("font %q after Enter", h.a.bodyFont.Name)
	}
}

// A click elsewhere closes the list and keeps the chosen font.
func TestComboClickAway(t *testing.T) {
	h := newHarness(t, "")
	c := h.openCombo(monoFamily(t))
	h.typeInCombo("mono")
	h.click(float32(h.size.X/2), 900)
	if c.open || c.field.Text() != defaultFontName || h.a.bodyFont.Name != "" {
		t.Errorf("open %v, field %q, font %q", c.open, c.field.Text(), h.a.bodyFont.Name)
	}
}

// The list fills in when the background scan ends, without a dialog.
func TestComboScanArrives(t *testing.T) {
	h := newHarness(t, "")
	h.a.scanFonts()
	h.waitFor(fontsScanned)
	if c := h.a.fontBox; c.names == nil || c.names[0] != defaultFontName {
		t.Errorf("names %q", c.names)
	}
}

func fontsScanned(a *App) bool { return a.families != nil }

func TestSavedFontForCommandLine(t *testing.T) {
	h := newHarness(t, "")
	h.chooseFont("Test Mono", monoFamily(t))
	set, err := SavedFont()
	if err != nil || set.Name != "Test Mono" || set.TTF[sysfont.Regular] == nil {
		t.Errorf("SavedFont = %q, %v", set.Name, err)
	}
}

// A path too long for the toolbar shows as the file name.
func TestDocumentNameFits(t *testing.T) {
	h := newHarness(t, "")
	h.a.path = "/some/very/long/folder/name/that/goes/on/chapter.md"
	gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}}
	if got := h.a.documentName(gtx, 100); got != "chapter.md" {
		t.Errorf("name %q in a narrow space", got)
	}
	if got := h.a.documentName(gtx, 5000); got != h.a.path {
		t.Errorf("name %q with room to spare", got)
	}
}

// The box sits in the toolbar at the preview's left edge, and follows the
// divider; without a preview it hides.
func TestComboAboveThePreview(t *testing.T) {
	h := newHarness(t, "")
	want := int(h.a.split*float32(h.size.X)) + 12 + 20 // divider, and 10 dp in
	if h.a.fontBox.x != want {
		t.Errorf("box at x=%d, want %d", h.a.fontBox.x, want)
	}
	h.a.split = 0.4
	h.frame()
	if want := int(0.4*float32(h.size.X)) + 12 + 20; h.a.fontBox.x != want {
		t.Errorf("box at x=%d after moving the divider, want %d", h.a.fontBox.x, want)
	}
	h.a.fontBox.drop()
	h.a.view = viewEditor
	h.frame()
	if h.a.fontBox.open {
		t.Error("the list stays open without a preview")
	}
}
