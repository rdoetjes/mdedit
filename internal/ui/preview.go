package ui

import (
	"image"
	"image/color"
	"math"
	"os"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/image/math/fixed"

	"mdedit/internal/mdmodel"
	"mdedit/internal/render"
)

// Preview renders a parsed document.
type Preview struct {
	doc     *mdmodel.Document
	BaseDir string
	// OnLink is called when a link is clicked.
	OnLink func(url string)

	list   widget.List
	cache  []*render.Layout
	cacheW int
	cacheE float32
	clicks []gesture.Click
	text   *textPainter
	images map[string]*imageEntry
}

func NewPreview(shaper *text.Shaper) *Preview {
	p := &Preview{
		text:   newTextPainter(shaper),
		images: map[string]*imageEntry{},
	}
	p.list.Axis = layout.Vertical
	return p
}

// SetDocument replaces the displayed document, keeping the scroll position.
func (p *Preview) SetDocument(doc *mdmodel.Document) {
	p.doc = doc
	p.cache = make([]*render.Layout, len(doc.Blocks))
	for len(p.clicks) < len(doc.Blocks) {
		p.clicks = append(p.clicks, gesture.Click{})
	}
}

// ScrollToBlock scrolls so block i is at the top.
func (p *Preview) ScrollToBlock(i int) {
	p.list.Position.First = i + 1 // index 0 is the top padding
	p.list.Position.Offset = 0
	p.list.Position.BeforeEnd = true
}

// EnsureVisible scrolls to block i if it is outside the visible range.
func (p *Preview) EnsureVisible(i int) {
	pos := p.list.Position
	first, last := pos.First-1, pos.First+pos.Count-2
	if pos.Count == 0 || i < first || i > last {
		p.ScrollToBlock(i)
	}
}

const maxContentWidth = unit.Dp(880)

func (p *Preview) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	paint.FillShape(gtx.Ops, render.Light.Bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
	if p.doc == nil {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
	width := gtx.Constraints.Max.X
	pad := gtx.Dp(32)
	if width < gtx.Dp(500) {
		pad = gtx.Dp(12)
	}
	contentW := min(width-2*pad-gtx.Dp(10), gtx.Dp(maxContentWidth)) // leave room for the scrollbar
	contentW = max(contentW, gtx.Dp(100))
	ox := float32((width - contentW) / 2)

	em := float32(gtx.Sp(16))
	if contentW != p.cacheW || em != p.cacheE {
		p.cache = make([]*render.Layout, len(p.doc.Blocks))
		p.cacheW, p.cacheE = contentW, em
	}
	ctx := &render.Ctx{
		Em:      em,
		Theme:   &render.Light,
		Measure: p.text.measure,
		ImageSize: func(src string) (int, int, bool) {
			if img := p.image(src); img != nil {
				return img.size.X, img.size.Y, true
			}
			return 0, 0, false
		},
	}

	n := len(p.doc.Blocks)
	return material.List(th, &p.list).Layout(gtx, n+2, func(gtx layout.Context, i int) layout.Dimensions {
		if i == 0 || i == n+1 {
			return layout.Dimensions{Size: image.Pt(width, pad)}
		}
		bi := i - 1
		l := p.cache[bi]
		if l == nil {
			var prev *mdmodel.Block
			if bi > 0 {
				prev = &p.doc.Blocks[bi-1]
			}
			l = ctx.Block(&p.doc.Blocks[bi], prev, float32(contentW))
			p.cache[bi] = l
		}
		size := image.Pt(width, int(math.Ceil(float64(l.Height))))

		for {
			ev, ok := p.clicks[bi].Update(gtx.Source)
			if !ok {
				break
			}
			if ev.Kind != gesture.KindClick {
				continue
			}
			pos := f32.Pt(float32(ev.Position.X)-ox, float32(ev.Position.Y))
			for _, pr := range l.Prims {
				if pr.Kind == render.PLink && pos.X >= pr.X && pos.X < pr.X+pr.W && pos.Y >= pr.Y && pos.Y < pr.Y+pr.H {
					if p.OnLink != nil {
						p.OnLink(pr.URL)
					}
					break
				}
			}
		}

		p.paint(gtx, l, ox)
		area := clip.Rect{Max: size}.Push(gtx.Ops)
		p.clicks[bi].Add(gtx.Ops)
		area.Pop()
		return layout.Dimensions{Size: size}
	})
}

func rnd(v float32) int { return int(math.Round(float64(v))) }

func (p *Preview) paint(gtx layout.Context, l *render.Layout, ox float32) {
	ops := gtx.Ops
	for _, pr := range l.Prims {
		x := pr.X + ox
		switch pr.Kind {
		case render.PRect:
			r := image.Rect(rnd(x), rnd(pr.Y), rnd(x+pr.W), rnd(pr.Y+pr.H))
			rr := clip.UniformRRect(r, rnd(pr.Radius))
			if pr.Stroke > 0 {
				paint.FillShape(ops, pr.Color, clip.Stroke{Path: rr.Path(ops), Width: pr.Stroke}.Op())
			} else {
				paint.FillShape(ops, pr.Color, rr.Op(ops))
			}
		case render.PEllipse:
			e := clip.Ellipse{Min: image.Pt(rnd(x), rnd(pr.Y)), Max: image.Pt(rnd(x+pr.W), rnd(pr.Y+pr.H))}
			if pr.Stroke > 0 {
				paint.FillShape(ops, pr.Color, clip.Stroke{Path: e.Path(ops), Width: pr.Stroke}.Op())
			} else {
				paint.FillShape(ops, pr.Color, e.Op(ops))
			}
		case render.PLine:
			var path clip.Path
			path.Begin(ops)
			path.MoveTo(f32.Pt(x, pr.Y))
			path.LineTo(f32.Pt(pr.W+ox, pr.H))
			paint.FillShape(ops, pr.Color, clip.Stroke{Path: path.End(), Width: pr.Stroke}.Op())
		case render.PText:
			p.text.draw(gtx, pr.Font, pr.Size, pr.Text, x, pr.Y, pr.Color)
		case render.PImage:
			if img := p.image(pr.URL); img != nil {
				st := op.Affine(f32.Affine2D{}.
					Scale(f32.Point{}, f32.Pt(pr.W/float32(img.size.X), pr.H/float32(img.size.Y))).
					Offset(f32.Pt(float32(rnd(x)), float32(rnd(pr.Y))))).Push(ops)
				img.op.Add(ops)
				cl := clip.Rect{Max: img.size}.Push(ops)
				paint.PaintOp{}.Add(ops)
				cl.Pop()
				st.Pop()
			}
		case render.PLink:
			area := clip.Rect(image.Rect(rnd(x), rnd(pr.Y), rnd(x+pr.W), rnd(pr.Y+pr.H))).Push(ops)
			pointer.CursorPointer.Add(ops)
			area.Pop()
		}
	}
}

type imageEntry struct {
	mod  int64
	op   paint.ImageOp
	size image.Point
}

// image loads and caches the image referenced by src.
func (p *Preview) image(src string) *imageEntry {
	path, ok := render.ResolveImage(p.BaseDir, src)
	if !ok {
		return nil
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil
	}
	if e, ok := p.images[path]; ok && e.mod == st.ModTime().UnixNano() {
		return e
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		p.images[path] = nil
		return nil
	}
	e := &imageEntry{mod: st.ModTime().UnixNano(), op: paint.NewImageOp(img), size: img.Bounds().Size()}
	p.images[path] = e
	return e
}

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
	var m render.Metrics
	gs := t.shape(f, size, s)
	var adv fixed.Int26_6
	for _, g := range gs {
		adv += g.Advance
	}
	if len(gs) > 0 {
		m = render.Metrics{
			Width:   float32(adv) / 64,
			Ascent:  float32(gs[0].Ascent) / 64,
			Descent: float32(gs[0].Descent) / 64,
		}
	}
	if len(t.cache) > 100000 {
		t.cache = map[measureKey]render.Metrics{}
	}
	t.cache[k] = m
	return m
}

func (t *textPainter) draw(gtx layout.Context, f render.FontSpec, size float32, s string, x, baseline float32, col color.NRGBA) {
	gs := t.shape(f, size, s)
	if len(gs) == 0 {
		return
	}
	ops := gtx.Ops
	// Shape draws relative to the first glyph's origin on the baseline.
	st := op.Offset(image.Pt(rnd(x)+gs[0].X.Floor(), rnd(baseline))).Push(ops)
	outline := clip.Outline{Path: t.shaper.Shape(gs)}.Op().Push(ops)
	paint.ColorOp{Color: col}.Add(ops)
	paint.PaintOp{}.Add(ops)
	outline.Pop()
	if call := t.shaper.Bitmaps(gs); call != (op.CallOp{}) {
		call.Add(ops)
	}
	st.Pop()
}
