package mdmodel

import "testing"

func TestMathParsing(t *testing.T) {
	cases := []struct {
		src   string
		kinds []Kind
		math  []string // TeX of math spans/blocks, in order
	}{
		{"$$\n\\frac{a}{b}\n$$\n", []Kind{Math}, []string{`\frac{a}{b}`}},
		{"$$x^2$$\n", []Kind{Math}, []string{"x^2"}},
		{"$$\na\n\\\\\nb\n$$\n\nafter", []Kind{Math, Paragraph}, []string{"a\n\\\\\nb"}},
		{"text $a_b_c$ and $x*y*z$ end", []Kind{Paragraph}, []string{"a_b_c", "x*y*z"}},
		{"costs $5 and $10 today", []Kind{Paragraph}, nil},
		{"a $ b $ c", []Kind{Paragraph}, nil},
		{"escaped \\$x$ not maths", []Kind{Paragraph}, nil},
		{"`$x$` is code", []Kind{Paragraph}, nil},
		{"para\n$$\ny\n$$", []Kind{Paragraph, Math}, []string{"y"}},
		{"```math\nE = mc^2\n```", []Kind{Math}, []string{"E = mc^2"}},
		{"$a$$b$", []Kind{Paragraph}, []string{"a", "b"}},
		{"inline $$\\sum x$$ display", []Kind{Paragraph}, []string{`\sum x`}},
	}
	for _, c := range cases {
		doc := Parse([]byte(c.src))
		var kinds []Kind
		var maths []string
		for _, b := range doc.Blocks {
			kinds = append(kinds, b.Kind)
			for _, s := range b.Spans {
				if s.Math {
					maths = append(maths, s.Text)
				}
			}
		}
		if len(kinds) != len(c.kinds) {
			t.Errorf("%q: kinds %v, want %v", c.src, kinds, c.kinds)
			continue
		}
		for i := range kinds {
			if kinds[i] != c.kinds[i] {
				t.Errorf("%q: kinds %v, want %v", c.src, kinds, c.kinds)
			}
		}
		if len(maths) != len(c.math) {
			t.Errorf("%q: maths %q, want %q", c.src, maths, c.math)
			continue
		}
		for i := range maths {
			if maths[i] != c.math[i] {
				t.Errorf("%q: maths %q, want %q", c.src, maths, c.math)
			}
		}
	}
}
