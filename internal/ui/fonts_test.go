//go:build !js

package ui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gioui.org/layout"
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
	h.run(newCommand("", "", 0, h.a.cmdFonts))
	d := h.a.dialog.(*FontDialog)
	d.selected = name
	d.done(d.selected)
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
	if h.a.dialog != nil || !h.a.focusEditor && !h.editorFocused() {
		t.Error("the dialog should close and give the editor the focus back")
	}
	if !h.restart().bodyIsMono() {
		t.Error("the font is not restored at the next start")
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

func TestFontDialogFilter(t *testing.T) {
	d := newFontDialog(defaultFontName, nil)
	if d.names != nil {
		t.Fatal("no fonts should be listed before the scan ends")
	}
	d.setFamilies([]sysfont.Family{{Name: "Arial"}, {Name: "Georgia"}, {Name: "Arial Black"}})
	d.filter.SetText("ARIAL")
	d.applyFilter()
	if !slices.Equal(d.shown, []string{"Arial", "Arial Black"}) {
		t.Errorf("shown %q", d.shown)
	}
}

// The dialog fills in when the background scan ends.
func TestFontScanArrives(t *testing.T) {
	h := newHarness(t, "")
	h.a.cmdFonts(layout.Context{})
	h.waitFor(fontsScanned)
	if d := h.a.dialog.(*FontDialog); d.names == nil || d.names[0] != defaultFontName {
		t.Errorf("names %q", d.names)
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
