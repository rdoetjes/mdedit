package ui

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"

	"mdedit/internal/render"
)

// textPainter measures and draws single-line strings with the Gio shaper,
// using identical parameters for both so layout and painting agree.
type textPainter struct {
	shaper *text.Shaper
	cache  map[measureKey]render.Metrics
	glyphs []text.Glyph
}

type measureKey struct {
	f    render.FontSpec
	size float32
	s    string
}

const maxMeasureCache = 100000

func newTextPainter(s *text.Shaper) *textPainter {
	return &textPainter{shaper: s, cache: map[measureKey]render.Metrics{}}
}

func giofont(f render.FontSpec) font.Font {
	ft := font.Font{Typeface: "Go"}
	if f.Mono {
		ft.Typeface = "Go Mono"
	}
	if f.Bold {
		ft.Weight = font.Bold
	}
	if f.Italic {
		ft.Style = font.Italic
	}
	return ft
}

func (t *textPainter) shape(f render.FontSpec, size float32, s string) []text.Glyph {
	t.shaper.LayoutString(text.Parameters{
		Font:     giofont(f),
		PxPerEm:  fixed.Int26_6(size * 64),
		MaxWidth: 1 << 24,
	}, s)
	t.glyphs = t.glyphs[:0]
	for g, ok := t.shaper.NextGlyph(); ok; g, ok = t.shaper.NextGlyph() {
		t.glyphs = append(t.glyphs, g)
	}
	return t.glyphs
}

func (t *textPainter) measure(f render.FontSpec, size float32, s string) render.Metrics {
	k := measureKey{f, size, s}
	if m, ok := t.cache[k]; ok {
		return m
	}
	m := metricsOf(t.shape(f, size, s))
	if len(t.cache) > maxMeasureCache {
		t.cache = map[measureKey]render.Metrics{}
	}
	t.cache[k] = m
	return m
}

func metricsOf(gs []text.Glyph) render.Metrics {
	if len(gs) == 0 {
		return render.Metrics{}
	}
	var adv fixed.Int26_6
	for _, g := range gs {
		adv += g.Advance
	}
	return render.Metrics{
		Width:   float32(adv) / 64,
		Ascent:  float32(gs[0].Ascent) / 64,
		Descent: float32(gs[0].Descent) / 64,
	}
}

func (t *textPainter) draw(gtx layout.Context, f render.FontSpec, size float32, s string, x, baseline float32, c color.NRGBA) {
	gs := t.shape(f, size, s)
	if len(gs) == 0 {
		return
	}
	ops := gtx.Ops
	// Shape draws relative to the first glyph's origin on the baseline.
	st := op.Offset(image.Pt(rnd(x)+gs[0].X.Floor(), rnd(baseline))).Push(ops)
	outline := clip.Outline{Path: t.shaper.Shape(gs)}.Op().Push(ops)
	paint.ColorOp{Color: c}.Add(ops)
	paint.PaintOp{}.Add(ops)
	outline.Pop()
	if call := t.shaper.Bitmaps(gs); call != (op.CallOp{}) {
		call.Add(ops)
	}
	st.Pop()
}
