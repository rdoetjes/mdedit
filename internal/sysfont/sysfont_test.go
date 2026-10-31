package sysfont

import (
	"bytes"
	"testing"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"golang.org/x/image/font/gofont/goregular"
)

func loader(t *testing.T, ttf []byte) *ot.Loader {
	ld, err := ot.NewLoader(bytes.NewReader(ttf))
	if err != nil {
		t.Fatal(err)
	}
	return ld
}

// A rewritten font must keep every table and still parse.
func TestStandalone(t *testing.T) {
	ld := loader(t, goregular.TTF)
	out, err := standalone(ld)
	if err != nil {
		t.Fatal(err)
	}
	back := loader(t, out)
	for _, tag := range ld.Tables() {
		a, _ := ld.RawTable(tag)
		b, err := back.RawTable(tag)
		if err != nil || !bytes.Equal(a, b) {
			t.Errorf("table %s changed: %v", tag, err)
		}
	}
	if _, err := font.ParseTTF(bytes.NewReader(out)); err != nil {
		t.Error(err)
	}
}

func TestUsable(t *testing.T) {
	if !usable(loader(t, goregular.TTF)) {
		t.Error("the Go font should be usable")
	}
}

func TestUnicodeFormat4(t *testing.T) {
	// A format 4 subtable at offset 12, and records pointing at it.
	cmap := append(make([]byte, 12), 0, 4)
	tests := []struct {
		platform, encoding byte
		want               bool
	}{
		{3, 1, true},
		{0, 3, true},
		{3, 0, false}, // symbol
		{1, 0, false}, // old Mac
	}
	for _, tt := range tests {
		rec := []byte{0, tt.platform, 0, tt.encoding, 0, 0, 0, 12}
		if got := unicodeFormat4(cmap, rec); got != tt.want {
			t.Errorf("platform %d encoding %d: %v", tt.platform, tt.encoding, got)
		}
	}
}

func TestPickStyles(t *testing.T) {
	regular := candidate{Face{Path: "r"}, font.Aspect{Style: font.StyleNormal, Weight: font.WeightNormal, Stretch: font.StretchNormal}}
	light := candidate{Face{Path: "l"}, font.Aspect{Style: font.StyleNormal, Weight: font.WeightLight, Stretch: font.StretchNormal}}
	bold := candidate{Face{Path: "b"}, font.Aspect{Style: font.StyleNormal, Weight: font.WeightBold, Stretch: font.StretchNormal}}
	italic := candidate{Face{Path: "i"}, font.Aspect{Style: font.StyleItalic, Weight: font.WeightNormal, Stretch: font.StretchNormal}}
	f, ok := pickStyles("X", []candidate{light, bold, regular, italic})
	if !ok || f.Faces[Regular].Path != "r" || f.Faces[Bold].Path != "b" || f.Faces[Italic].Path != "i" || f.Faces[BoldItalic] != nil {
		t.Errorf("styles %+v", f)
	}
	if _, ok := pickStyles("Y", []candidate{italic, bold}); ok {
		t.Error("a family without an upright regular face should not count")
	}
}
