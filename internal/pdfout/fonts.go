package pdfout

import (
	"image/color"
	"path/filepath"
	"strings"

	"github.com/go-pdf/fpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"

	"mdedit/internal/render"
	"mdedit/internal/tex"
)

// goFace describes one of the eight embedded Go fonts.
type goFace struct {
	family, style string
	ttf           []byte
}

// goFaces are in the order of fontIndex.
var goFaces = []goFace{
	{"go", "", goregular.TTF},
	{"go", "B", gobold.TTF},
	{"go", "I", goitalic.TTF},
	{"go", "BI", gobolditalic.TTF},
	{"gomono", "", gomono.TTF},
	{"gomono", "B", gomonobold.TTF},
	{"gomono", "I", gomonoitalic.TTF},
	{"gomono", "BI", gomonobolditalic.TTF},
}

// goFonts are the parsed Go fonts, used to find characters they lack.
var goFonts = parseGoFonts()

func parseGoFonts() []*sfnt.Font {
	out := make([]*sfnt.Font, len(goFaces))
	for i, f := range goFaces {
		font, err := sfnt.Parse(f.ttf)
		if err != nil {
			panic(err)
		}
		out[i] = font
	}
	return out
}

func registerFonts(pdf *fpdf.Fpdf) {
	for _, f := range goFaces {
		pdf.AddUTF8FontFromBytes(f.family, f.style, f.ttf)
	}
}

func fontIndex(f render.FontSpec) int {
	i := 0
	if f.Bold {
		i++
	}
	if f.Italic {
		i += 2
	}
	if f.Mono {
		i += 4
	}
	return i
}

func (e *exporter) setFont(f render.FontSpec, size float32) {
	face := goFaces[fontIndex(f)]
	e.pdf.SetFont(face.family, face.style, float64(size))
}

func (e *exporter) measure(f render.FontSpec, size float32, s string) render.Metrics {
	e.setFont(f, size)
	var w float32
	for _, pc := range e.pieces(f, s) {
		w += e.pieceWidth(pc, size)
	}
	return render.Metrics{Width: w, Ascent: ascent * size, Descent: descent * size}
}

func (e *exporter) pieceWidth(pc piece, size float32) float32 {
	if !pc.fallback {
		return float32(e.pdf.GetStringWidth(pc.text))
	}
	var w float32
	for _, r := range pc.text {
		_, _, adv, _ := tex.Fallback(r)
		w += adv * size
	}
	return w
}

// Characters the Go fonts lack (⌘, ⇧, ✓ …) are drawn as outlines from the
// maths font instead of as empty boxes.

type piece struct {
	text     string
	fallback bool
}

type coverKey struct {
	f render.FontSpec
	r rune
}

// pieces splits s into runs the Go font can set and runs that need the
// fallback font.
func (e *exporter) pieces(f render.FontSpec, s string) []piece {
	var out []piece
	for _, r := range s {
		fb := e.needsFallback(f, r)
		if n := len(out); n > 0 && out[n-1].fallback == fb {
			out[n-1].text += string(r)
		} else {
			out = append(out, piece{string(r), fb})
		}
	}
	return out
}

// needsFallback reports whether r is missing from the Go font but available
// in the fallback font. Otherwise the PDF font shows its missing-glyph box.
func (e *exporter) needsFallback(f render.FontSpec, r rune) bool {
	if e.covered(f, r) {
		return false
	}
	_, _, _, ok := tex.Fallback(r)
	return ok
}

func (e *exporter) covered(f render.FontSpec, r rune) bool {
	if r < 0x80 {
		return true
	}
	k := coverKey{f, r}
	if v, ok := e.coverage[k]; ok {
		return v
	}
	var buf sfnt.Buffer
	g, err := goFonts[fontIndex(f)].GlyphIndex(&buf, r)
	e.coverage[k] = err == nil && g != 0
	return e.coverage[k]
}

// drawText sets s with its baseline at (x, y), switching to the fallback
// font where needed.
func (e *exporter) drawText(f render.FontSpec, size float32, s string, x, y float32, col color.NRGBA) {
	e.setFont(f, size)
	e.pdf.SetTextColor(int(col.R), int(col.G), int(col.B))
	for _, pc := range e.pieces(f, s) {
		if pc.fallback {
			x = e.drawFallback(pc.text, size, x, y, col)
			continue
		}
		e.pdf.Text(float64(x), float64(y), pc.text)
		x += float32(e.pdf.GetStringWidth(pc.text))
	}
}

// drawFallback draws characters from the fallback font and returns the x
// after them.
func (e *exporter) drawFallback(s string, size, x, y float32, col color.NRGBA) float32 {
	for _, r := range s {
		face, g, adv, _ := tex.Fallback(r)
		e.fillGlyph(face, g, x, y, size, size, col)
		x += adv * size
	}
	return x
}

// outlineTransform maps glyph outline coordinates (em, y down) to the page.
type outlineTransform struct {
	x, y, sx, sy float32
}

func (t outlineTransform) pt(q [2]float32) (float64, float64) {
	return float64(t.x + q[0]*t.sx), float64(t.y + q[1]*t.sy)
}

// fillGlyph fills a glyph outline with its origin at (x, y) in page
// coordinates, scaled by sx and sy.
func (e *exporter) fillGlyph(face uint8, g uint16, x, y, sx, sy float32, col color.NRGBA) {
	segs := tex.Outline(face, g)
	if len(segs) == 0 {
		return
	}
	t := outlineTransform{x, y, sx, sy}
	e.setFill(col)
	for i, s := range segs {
		e.pathSegment(s, t, i > 0)
	}
	e.pdf.ClosePath()
	e.pdf.DrawPath("F")
}

// pathSegment adds one outline segment; a MoveTo after the first segment
// closes the previous contour.
func (e *exporter) pathSegment(s tex.Seg, t outlineTransform, closePrev bool) {
	switch s.Op {
	case tex.MoveTo:
		if closePrev {
			e.pdf.ClosePath()
		}
		e.pdf.MoveTo(t.pt(s.P[0]))
	case tex.LineTo:
		e.pdf.LineTo(t.pt(s.P[0]))
	case tex.QuadTo:
		cx, cy := t.pt(s.P[0])
		px, py := t.pt(s.P[1])
		e.pdf.CurveTo(cx, cy, px, py)
	case tex.CubeTo:
		e.cubeTo(s, t)
	}
}

func (e *exporter) cubeTo(s tex.Seg, t outlineTransform) {
	c1x, c1y := t.pt(s.P[0])
	c2x, c2y := t.pt(s.P[1])
	px, py := t.pt(s.P[2])
	e.pdf.CurveBezierCubicTo(c1x, c1y, c2x, c2y, px, py)
}

func imageType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "PNG"
	case ".jpg", ".jpeg":
		return "JPG"
	case ".gif":
		return "GIF"
	}
	return ""
}

func supportedImage(path string) bool { return imageType(path) != "" }
