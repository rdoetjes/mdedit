package mdmodel

import (
	"os"
	"testing"
)

func TestParseTestFile(t *testing.T) {
	src, err := os.ReadFile("testdata/TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := Parse(src)
	if len(doc.Blocks) == 0 {
		t.Fatal("no blocks")
	}
	if doc.HeadingIndex("overview") < 0 {
		t.Error("missing #overview anchor")
	}
	for i, b := range doc.Blocks {
		if testing.Verbose() {
			t.Logf("%3d L%-3d gap=%.2f kind=%d decor=%+v %q", i, b.Line, b.Gap, b.Kind, b.Decor, spanText(b.Spans))
		}
	}
}

func spanText(s []Span) string {
	var out string
	for _, x := range s {
		out += x.Text
	}
	if len(out) > 50 {
		out = out[:50]
	}
	return out
}
