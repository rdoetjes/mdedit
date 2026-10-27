package pdfout

import (
	"os"
	"path/filepath"
	"testing"

	"mdedit/internal/mdmodel"
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
