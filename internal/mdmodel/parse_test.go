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

func TestVideoLink(t *testing.T) {
	src := "[![My talk](https://img.youtube.com/vi/dQw4w9WgXcQ/maxresdefault.jpg)](https://www.youtube.com/watch?v=dQw4w9WgXcQ)\n\n" +
		"[![Logo](logo.png)](https://example.com)\n\nText [![x](y.png)](z) inline.\n"
	doc := Parse([]byte(src))
	if len(doc.Blocks) != 3 {
		t.Fatalf("got %d blocks", len(doc.Blocks))
	}
	v := doc.Blocks[0]
	if v.Kind != Image || !v.Video || v.Alt != "My talk" || v.Link != "https://www.youtube.com/watch?v=dQw4w9WgXcQ" {
		t.Errorf("video block: %+v", v)
	}
	if l := doc.Blocks[1]; l.Kind != Image || l.Video || l.Link != "https://example.com" {
		t.Errorf("linked image: %+v", l)
	}
	if doc.Blocks[2].Kind != Paragraph {
		t.Errorf("image link inside text should stay inline: %+v", doc.Blocks[2])
	}
}
