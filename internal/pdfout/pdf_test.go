package pdfout

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-pdf/fpdf"

	"mdedit/internal/mdmodel"
	"mdedit/internal/render"
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
	e := &exporter{pdf: fpdf.New("P", "pt", "A4", ""), coverage: map[coverKey]bool{}}
	registerFonts(e.pdf)
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
