package mdmodel

import (
	"slices"
	"testing"
)

type mathCase struct {
	src   string
	kinds []Kind
	math  []string // TeX of math spans/blocks, in order
}

var mathCases = []mathCase{
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

func TestMathParsing(t *testing.T) {
	for _, c := range mathCases {
		kinds, maths := mathSummary(Parse([]byte(c.src)))
		if !slices.Equal(kinds, c.kinds) {
			t.Errorf("%q: kinds %v, want %v", c.src, kinds, c.kinds)
		}
		if !slices.Equal(maths, c.math) {
			t.Errorf("%q: maths %q, want %q", c.src, maths, c.math)
		}
	}
}

// mathSummary lists the kinds of the blocks and the TeX of all maths.
func mathSummary(doc *Document) ([]Kind, []string) {
	var kinds []Kind
	var maths []string
	for _, b := range doc.Blocks {
		kinds = append(kinds, b.Kind)
		maths = append(maths, mathTexts(b.Spans)...)
	}
	return kinds, maths
}

func mathTexts(spans []Span) []string {
	var out []string
	for _, s := range spans {
		if s.Math {
			out = append(out, s.Text)
		}
	}
	return out
}
