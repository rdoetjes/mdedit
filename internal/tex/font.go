package tex

import (
	_ "embed"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// STIX Two Math (SIL Open Font License, see fonts/OFL.txt) supplies the
// maths glyphs, including the Unicode mathematical alphabets.
//
//go:embed fonts/STIXTwoMath-Regular.ttf
var stixMath []byte

// Faces. Glyphs missing from the maths font fall back to Go Regular.
const (
	FaceMath uint8 = iota
	FaceFallback
)

type face struct {
	f    *sfnt.Font
	upem fixed.Int26_6
	buf  sfnt.Buffer
}

var (
	fontMu  sync.Mutex
	faces   [2]*face
	metrics = map[glyphKey]glyphMetrics{}
	paths   = map[glyphKey][]Seg{}
)

type glyphKey struct {
	face  uint8
	glyph uint16
}

// glyphMetrics are in em, with y pointing down.
type glyphMetrics struct {
	adv                    float32
	minX, minY, maxX, maxY float32
}

func loadFaces() {
	if faces[0] != nil {
		return
	}
	for i, data := range [][]byte{stixMath, goregular.TTF} {
		f, err := sfnt.Parse(data)
		if err != nil {
			panic("tex: bad embedded font: " + err.Error())
		}
		faces[i] = &face{f: f, upem: fixed.I(int(f.UnitsPerEm()))}
	}
}

// lookup finds the face and glyph for r.
func lookup(r rune) (uint8, uint16, bool) {
	fontMu.Lock()
	defer fontMu.Unlock()
	loadFaces()
	for i, fc := range faces {
		if g, err := fc.f.GlyphIndex(&fc.buf, r); err == nil && g != 0 {
			return uint8(i), uint16(g), true
		}
	}
	return 0, 0, false
}

func glyphInfo(faceID uint8, g uint16) glyphMetrics {
	k := glyphKey{faceID, g}
	fontMu.Lock()
	defer fontMu.Unlock()
	if m, ok := metrics[k]; ok {
		return m
	}
	loadFaces()
	fc := faces[faceID]
	b, adv, err := fc.f.GlyphBounds(&fc.buf, sfnt.GlyphIndex(g), fc.upem, font.HintingNone)
	var m glyphMetrics
	if err == nil {
		u := float32(fc.upem)
		m = glyphMetrics{
			adv:  float32(adv) / u,
			minX: float32(b.Min.X) / u, minY: float32(b.Min.Y) / u,
			maxX: float32(b.Max.X) / u, maxY: float32(b.Max.Y) / u,
		}
	}
	metrics[k] = m
	return m
}

// SegOp is a path operation.
type SegOp uint8

const (
	MoveTo SegOp = iota
	LineTo
	QuadTo
	CubeTo
)

// Seg is one segment of a glyph outline, in em with y pointing down and the
// origin on the baseline. P holds 1 (move, line), 2 (quad) or 3 (cube)
// points.
type Seg struct {
	Op SegOp
	P  [3][2]float32
}

// Outline returns the outline of a glyph. Renderers fill it with the
// non-zero winding rule.
func Outline(faceID uint8, g uint16) []Seg {
	k := glyphKey{faceID, g}
	fontMu.Lock()
	defer fontMu.Unlock()
	if p, ok := paths[k]; ok {
		return p
	}
	loadFaces()
	fc := faces[faceID]
	segs, err := fc.f.LoadGlyph(&fc.buf, sfnt.GlyphIndex(g), fc.upem, nil)
	var out []Seg
	if err == nil {
		u := float32(fc.upem)
		pt := func(p fixed.Point26_6) [2]float32 { return [2]float32{float32(p.X) / u, float32(p.Y) / u} }
		for _, s := range segs {
			var o Seg
			switch s.Op {
			case sfnt.SegmentOpMoveTo:
				o = Seg{Op: MoveTo, P: [3][2]float32{pt(s.Args[0])}}
			case sfnt.SegmentOpLineTo:
				o = Seg{Op: LineTo, P: [3][2]float32{pt(s.Args[0])}}
			case sfnt.SegmentOpQuadTo:
				o = Seg{Op: QuadTo, P: [3][2]float32{pt(s.Args[0]), pt(s.Args[1])}}
			case sfnt.SegmentOpCubeTo:
				o = Seg{Op: CubeTo, P: [3][2]float32{pt(s.Args[0]), pt(s.Args[1]), pt(s.Args[2])}}
			}
			out = append(out, o)
		}
	}
	paths[k] = out
	return out
}

// Fallback returns a glyph for r from the maths font, for text whose own
// font lacks it, with its advance in em.
func Fallback(r rune) (faceID uint8, g uint16, adv float32, ok bool) {
	faceID, g, ok = lookup(r)
	if !ok {
		return 0, 0, 0, false
	}
	return faceID, g, glyphInfo(faceID, g).adv, true
}
