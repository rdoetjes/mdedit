package sysfont

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/go-text/typesetting/font"
)

// TestRealFonts runs against the fonts on this computer: every family must
// load, parse, and go into a PDF. Set MDEDIT_FONTS to a folder to run it;
// a PDF per family is written there, to look at.
func TestRealFonts(t *testing.T) {
	dir := os.Getenv("MDEDIT_FONTS")
	if dir == "" {
		t.Skip("MDEDIT_FONTS not set")
	}
	start := time.Now()
	fams := Scan()
	t.Logf("%d families in %v", len(fams), time.Since(start).Round(time.Millisecond))
	bad := 0
	for _, f := range fams {
		if err := tryFamily(f, dir); err != nil {
			bad++
			t.Errorf("%s: %v", f.Name, err)
		}
	}
	t.Logf("%d failed", bad)
}

func tryFamily(f Family, dir string) error {
	s, err := f.Load()
	if err != nil {
		return err
	}
	for _, ttf := range s.TTF {
		if ttf == nil {
			continue
		}
		if _, err := font.ParseTTF(bytes.NewReader(ttf)); err != nil {
			return fmt.Errorf("parse: %w", err)
		}
	}
	return writePDF(s, filepath.Join(dir, f.Name+".pdf"))
}

var sample = "Hamburgefonstiv ÄÖÜ äöü éèê ß fi fl “quotes” — 0123456789"

func writePDF(s Set, path string) (err error) {
	defer recoverPDF(&err)
	pdf := fpdf.New("P", "pt", "A4", "")
	pdf.AddPage()
	y := 80.0
	for i, style := range []string{"", "B", "I", "BI"} {
		if s.TTF[i] == nil {
			continue
		}
		pdf.AddUTF8FontFromBytes("body", style, s.TTF[i])
		pdf.SetFont("body", style, 22)
		pdf.Text(50, y, sample)
		y += 40
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func recoverPDF(err *error) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("panic: %v", r)
	}
}
