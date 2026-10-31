package pdfout

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-pdf/fpdf"
	"golang.org/x/image/font/gofont/gomono"

	"mdedit/internal/mdmodel"
	"mdedit/internal/render"
	"mdedit/internal/sysfont"
	"mdedit/internal/tex"
)

func TestExportTestFile(t *testing.T) {
	src, err := os.ReadFile("../mdmodel/testdata/TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "TEST.pdf")
	if dir := os.Getenv("MDEDIT_PDF_OUT"); dir != "" {
		out = filepath.Join(dir, "TEST.pdf")
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := Write(f, mdmodel.Parse(src), Options{Title: "TEST"}); err != nil {
		t.Fatal(err)
	}
	st, _ := f.Stat()
	if st.Size() < 1000 {
		t.Fatalf("suspiciously small PDF: %d bytes", st.Size())
	}
}

// Characters missing from the Go fonts (⌘, ⇧, ⌫ …) are drawn from the maths
// font instead of showing as empty boxes, and measured at that width.
func TestFallbackGlyphs(t *testing.T) {
	fonts, err := newFonts(sysfont.Set{})
	if err != nil {
		t.Fatal(err)
	}
	e := &exporter{pdf: fpdf.New("P", "pt", "A4", ""), coverage: map[coverKey]bool{}, fonts: fonts}
	registerFonts(e.pdf, fonts)
	f := render.FontSpec{}
	got := e.pieces(f, "Press ⌘⇧S or ⌘⌫ now")
	want := []piece{{"Press ", false}, {"⌘⇧", true}, {"S or ", false}, {"⌘⌫", true}, {" now", false}}
	if len(got) != len(want) {
		t.Fatalf("pieces: %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("piece %d: %+v, want %+v", i, got[i], want[i])
		}
	}
	// Arrows exist in the Go font and stay as text.
	if p := e.pieces(f, "←→"); len(p) != 1 || p[0].fallback {
		t.Errorf("arrows: %+v", p)
	}
	_, _, adv, _ := tex.Fallback('⌘')
	if w := e.measure(f, 10, "⌘").Width; w != adv*10 || w <= 0 {
		t.Errorf("width of ⌘: %v, want %v", w, adv*10)
	}

	var buf bytes.Buffer
	doc := mdmodel.Parse([]byte("Use ⌘S to save, ⇧⌘S to save as, ⌘⌫ to delete."))
	if err := Write(&buf, doc, Options{}); err != nil {
		t.Fatal(err)
	}
}

// A chosen font replaces the text faces; code stays in Go Mono. Go Mono as
// the "chosen" font shows it: all letters are equally wide.
func TestChosenFont(t *testing.T) {
	fonts, err := newFonts(sysfont.Set{Name: "test", TTF: [4][]byte{gomono.TTF}})
	if err != nil {
		t.Fatal(err)
	}
	e := &exporter{pdf: fpdf.New("P", "pt", "A4", ""), coverage: map[coverKey]bool{}, fonts: fonts}
	registerFonts(e.pdf, fonts)
	body := render.FontSpec{}
	if a, b := e.measure(body, 10, "iiii"), e.measure(body, 10, "MMMM"); a.Width != b.Width {
		t.Errorf("widths %v and %v: the chosen font isn't used", a.Width, b.Width)
	}
	bold := render.FontSpec{Bold: true} // the font has no bold: its regular is used
	if a, b := e.measure(bold, 10, "iiii"), e.measure(bold, 10, "MMMM"); a.Width != b.Width {
		t.Errorf("bold widths %v and %v: a missing style isn't the regular face", a.Width, b.Width)
	}
	var buf bytes.Buffer
	opt := Options{Font: sysfont.Set{TTF: [4][]byte{gomono.TTF}}}
	if err := Write(&buf, mdmodel.Parse([]byte("# Title\n\nText **bold** `code` äöü.\n")), opt); err != nil {
		t.Fatal(err)
	}
}

// The Go fonts keep the metrics the layout has always used.
func TestGoFontMetrics(t *testing.T) {
	fonts, err := newFonts(sysfont.Set{})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fonts {
		if d := f.ascent - ascent; d > 0.0005 || d < -0.0005 {
			t.Errorf("%s %q: ascent %v, want %v", f.family, f.style, f.ascent, ascent)
		}
		if d := f.descent - descent; d > 0.0005 || d < -0.0005 {
			t.Errorf("%s %q: descent %v, want %v", f.family, f.style, f.descent, descent)
		}
	}
}
