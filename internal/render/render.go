// Package render lays out mdmodel blocks into simple positioned primitives
// (text runs, rectangles, lines, images and link areas).
//
// The layout is independent of the output device: the caller supplies the
// base font size and a text measuring function in its own units (pixels for
// the screen, points for PDF). The Gio preview and the PDF exporter then only
// have to paint primitives, so both outputs look the same.
package render

import (
	"image/color"

	"mdedit/internal/mdmodel"
	"mdedit/internal/tex"
)

// FontSpec selects one of the eight faces of the Go font family.
type FontSpec struct {
	Mono, Bold, Italic bool
}

// Metrics describes a measured string.
type Metrics struct {
	Width, Ascent, Descent float32
}

// Ctx supplies device-specific services to the layout.
type Ctx struct {
	// Em is the base font size in device units.
	Em float32
	// Measure measures s set in font f at the given size.
	Measure func(f FontSpec, size float32, s string) Metrics
	// ImageSize returns the pixel dimensions of the image at src, which is
	// resolved relative to the document. ok is false for images that can't
	// be shown.
	ImageSize func(src string) (w, h int, ok bool)
	// MaxImageHeight limits image height (for example to fit a PDF page).
	// Zero means no limit.
	MaxImageHeight float32
	Theme          *Theme
}

type PrimKind uint8

const (
	PRect PrimKind = iota
	PEllipse
	PLine
	PText
	PImage
	PLink
	PGlyph // a maths glyph outline from package tex
	PPoly  // a filled polygon
)

// Prim is a positioned drawing primitive. Coordinates are relative to the
// top-left corner of the block.
type Prim struct {
	Kind       PrimKind
	X, Y, W, H float32 // PText: Y is the baseline. PLine: (X,Y) to (W,H).
	Color      color.NRGBA
	Radius     float32 // PRect corner radius
	Stroke     float32 // outline width for PRect/PEllipse/PLine; 0 fills
	Text       string
	Font       FontSpec
	Size       float32
	URL        string // PLink target; PImage source
	// PGlyph: the glyph is drawn at (X, Y) on the baseline with em size
	// Size, stretched by XScale and YScale.
	Face           uint8
	Glyph          uint16
	XScale, YScale float32
	Points         [][2]float32 // PPoly
}

// Top returns the highest y coordinate the primitive covers.
func (p *Prim) Top() float32 {
	switch p.Kind {
	case PText:
		return p.Y - p.Size
	case PLine:
		return min(p.Y, p.H)
	case PGlyph:
		return p.Y - p.Size
	case PPoly:
		t := p.Points[0][1]
		for _, q := range p.Points {
			t = min(t, q[1])
		}
		return t
	}
	return p.Y
}

// Layout is a laid-out block.
type Layout struct {
	Prims  []Prim
	Height float32
	// Breaks are y offsets between lines or rows where the block may be split
	// across pages, in increasing order.
	Breaks []float32
}

// Theme holds the colours used by the layout.
type Theme struct {
	Text, Muted, Link, Border color.NRGBA
	CodeBg, InlineCodeBg      color.NRGBA
	TableAltBg, CheckFill, Bg color.NRGBA
	Error                     color.NRGBA
}

func rgb(c uint32) color.NRGBA {
	return color.NRGBA{R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c), A: 0xff}
}

// Light is a theme modelled on GitHub's Markdown styling.
var Light = Theme{
	Text:         rgb(0x1f2328),
	Muted:        rgb(0x59636e),
	Link:         rgb(0x0969da),
	Border:       rgb(0xd1d9e0),
	CodeBg:       rgb(0xf6f8fa),
	InlineCodeBg: rgb(0xeff1f3),
	TableAltBg:   rgb(0xf6f8fa),
	CheckFill:    rgb(0x0969da),
	Bg:           rgb(0xffffff),
	Error:        rgb(0xcf222e),
}

// Heading font sizes relative to the base size, as on GitHub.
var headingScale = [7]float32{1, 2, 1.5, 1.25, 1, 0.875, 0.85}

// codeScale is the size of monospaced text relative to the surrounding text.
const codeScale = 0.85

// Block lays out one block at the given width. prev is the preceding block,
// or nil, and decides whether block quote bars join up.
func (c *Ctx) Block(b, prev *mdmodel.Block, width float32) *Layout {
	em := c.Em
	gap := b.Gap * em
	th := c.Theme

	// Work out the content box from the containers.
	x0 := float32(0)
	type marker struct {
		x float32
		d mdmodel.Decor
	}
	var markers []marker
	type bar struct {
		x     float32
		joins bool
	}
	var bars []bar
	textColor := th.Text
	for i, d := range b.Decor {
		switch d.Kind {
		case mdmodel.Quote:
			joins := prev != nil && i < len(prev.Decor) && prev.Decor[i].ID == d.ID
			bars = append(bars, bar{x: x0, joins: joins})
			x0 += 0.25*em + em
			textColor = th.Muted
		case mdmodel.Item:
			if d.ShowMarker {
				markers = append(markers, marker{x: x0, d: d})
			}
			x0 += 2 * em
		}
	}
	cw := max(width-x0, 4*em)

	var content *Layout
	var firstBaseline float32
	switch b.Kind {
	case mdmodel.Heading:
		content, firstBaseline = c.heading(b, cw, textColor)
	case mdmodel.Code, mdmodel.HTML:
		content, firstBaseline = c.code(b, cw)
	case mdmodel.Rule:
		content = &Layout{Height: 0.25 * em, Prims: []Prim{{Kind: PRect, W: cw, H: 0.25 * em, Color: th.Border}}}
		firstBaseline = 0.25 * em
	case mdmodel.Table:
		content, firstBaseline = c.table(b.Table, cw, textColor)
	case mdmodel.Image:
		content, firstBaseline = c.image(b, cw, textColor)
	case mdmodel.Math:
		content, firstBaseline = c.mathBlock(b, cw, textColor)
	default:
		content, firstBaseline = c.paragraph(b.Spans, cw, textColor, b.Align)
	}

	out := &Layout{Height: gap + content.Height}
	for _, p := range content.Prims {
		p.X += x0
		p.Y += gap
		switch p.Kind {
		case PLine:
			p.W += x0
			p.H += gap
		case PPoly:
			pts := make([][2]float32, len(p.Points))
			for i, q := range p.Points {
				pts[i] = [2]float32{q[0] + x0, q[1] + gap}
			}
			p.Points = pts
		}
		out.Prims = append(out.Prims, p)
	}
	for _, br := range content.Breaks {
		out.Breaks = append(out.Breaks, br+gap)
	}
	firstBaseline += gap

	for _, br := range bars {
		top := gap
		if br.joins {
			top = 0
		}
		out.Prims = append(out.Prims, Prim{Kind: PRect, X: br.x, Y: top, W: 0.25 * em, H: out.Height - top, Color: th.Border})
	}
	for _, m := range markers {
		out.Prims = append(out.Prims, c.marker(m.x, firstBaseline, m.d, textColor)...)
	}
	return out
}

func (c *Ctx) marker(x, baseline float32, d mdmodel.Decor, col color.NRGBA) []Prim {
	em := c.Em
	right := x + 2*em - 0.4*em
	switch {
	case d.Task != mdmodel.NoTask:
		s := 0.85 * em
		bx, by := right-s, baseline-0.78*em
		if d.Task == mdmodel.TaskDone {
			return []Prim{
				{Kind: PRect, X: bx, Y: by, W: s, H: s, Radius: 0.15 * em, Color: c.Theme.CheckFill},
				{Kind: PLine, X: bx + 0.2*s, Y: by + 0.52*s, W: bx + 0.42*s, H: by + 0.74*s, Stroke: 0.12 * em, Color: c.Theme.Bg},
				{Kind: PLine, X: bx + 0.42*s, Y: by + 0.74*s, W: bx + 0.8*s, H: by + 0.28*s, Stroke: 0.12 * em, Color: c.Theme.Bg},
			}
		}
		return []Prim{{Kind: PRect, X: bx, Y: by, W: s, H: s, Radius: 0.15 * em, Stroke: 0.08 * em, Color: c.Theme.Muted}}
	case d.Ordered:
		m := c.Measure(FontSpec{}, em, d.Number)
		return []Prim{{Kind: PText, X: right - m.Width, Y: baseline, W: m.Width, Text: d.Number, Size: em, Color: col}}
	default:
		r := 0.18 * em
		cx, cy := right-r-0.1*em, baseline-0.32*em
		p := Prim{Kind: PEllipse, X: cx - r, Y: cy - r, W: 2 * r, H: 2 * r, Color: col}
		switch d.Bullet % 3 {
		case 1:
			p.Stroke = 0.07 * em
		case 2:
			p.Kind = PRect
		}
		return []Prim{p}
	}
}

func (c *Ctx) paragraph(spans []mdmodel.Span, w float32, col color.NRGBA, align mdmodel.Align) (*Layout, float32) {
	items := c.items(spans, c.Em, col, false)
	lines, h := c.layoutInline(items, w, c.Em, 1.5, false)
	l := &Layout{Height: h}
	c.emitLines(l, items, lines, 0, 0, w, align, false)
	return l, firstBaseline(lines, c.Em)
}

func (c *Ctx) heading(b *mdmodel.Block, w float32, col color.NRGBA) (*Layout, float32) {
	lvl := min(max(b.Level, 1), 6)
	size := headingScale[lvl] * c.Em
	if lvl == 6 {
		col = c.Theme.Muted
	}
	spans := make([]mdmodel.Span, len(b.Spans))
	for i, s := range b.Spans {
		s.Bold = true
		spans[i] = s
	}
	items := c.items(spans, size, col, false)
	lines, h := c.layoutInline(items, w, size, 1.25, false)
	l := &Layout{Height: h}
	c.emitLines(l, items, lines, 0, 0, w, b.Align, false)
	if lvl <= 2 {
		l.Height += 0.3 * c.Em
		l.Prims = append(l.Prims, Prim{Kind: PRect, Y: l.Height, W: w, H: max(1, c.Em/16), Color: c.Theme.Border})
		l.Height += max(1, c.Em/16)
	}
	return l, firstBaseline(lines, size)
}

func (c *Ctx) code(b *mdmodel.Block, w float32) (*Layout, float32) {
	em := c.Em
	pad := em
	size := codeScale * em
	items := c.items(b.Spans, size, c.Theme.Text, true)
	lines, h := c.layoutInline(items, w-2*pad, size, 1.45, true)
	l := &Layout{Height: h + 2*pad}
	l.Prims = append(l.Prims, Prim{Kind: PRect, W: w, H: l.Height, Radius: 0.375 * em, Color: c.Theme.CodeBg})
	c.emitLines(l, items, lines, pad, pad, w-2*pad, mdmodel.AlignNone, true)
	return l, pad + firstBaseline(lines, size)
}

func (c *Ctx) image(b *mdmodel.Block, w float32, col color.NRGBA) (*Layout, float32) {
	pw, ph, ok := 0, 0, false
	if c.ImageSize != nil {
		pw, ph, ok = c.ImageSize(b.Src)
	}
	if !ok || pw == 0 || ph == 0 {
		alt := b.Alt
		if alt == "" {
			alt = b.Src
		}
		return c.paragraph([]mdmodel.Span{{Text: "[image: " + alt + "]", Italic: true, Link: b.Src}}, w, col, b.Align)
	}
	// Treat image pixels as CSS pixels: 16 per em.
	iw := float32(pw) * c.Em / 16
	ih := float32(ph) * c.Em / 16
	if iw > w {
		ih *= w / iw
		iw = w
	}
	if c.MaxImageHeight > 0 && ih > c.MaxImageHeight {
		iw *= c.MaxImageHeight / ih
		ih = c.MaxImageHeight
	}
	x := float32(0)
	switch b.Align {
	case mdmodel.AlignCenter:
		x = (w - iw) / 2
	case mdmodel.AlignRight:
		x = w - iw
	}
	return &Layout{Height: ih, Prims: []Prim{{Kind: PImage, X: x, W: iw, H: ih, URL: b.Src}}}, min(ih, c.Em)
}

func (c *Ctx) table(t *mdmodel.TableData, w float32, col color.NRGBA) (*Layout, float32) {
	em := c.Em
	padX, padY := 13*em/16, 6*em/16
	border := max(1, em/16)
	ncol := len(t.Align)
	rows := make([][][]mdmodel.Span, 0, len(t.Rows)+1)
	rows = append(rows, t.Header)
	rows = append(rows, t.Rows...)
	cellItems := func(r, ci int) []inlineItem {
		if ci >= len(rows[r]) {
			return nil
		}
		spans := rows[r][ci]
		if r == 0 {
			spans = append([]mdmodel.Span(nil), spans...)
			for i := range spans {
				spans[i].Bold = true
			}
		}
		return c.items(spans, em, col, false)
	}

	// Column widths: natural widths, shrunk proportionally if too wide.
	natural := make([]float32, ncol)
	for r := range rows {
		for ci := 0; ci < ncol; ci++ {
			lines, _ := c.layoutInline(cellItems(r, ci), 1e9, em, 1.5, false)
			for _, ln := range lines {
				natural[ci] = max(natural[ci], ln.width+2*padX)
			}
			natural[ci] = max(natural[ci], 2*padX+em)
		}
	}
	var total float32
	for _, n := range natural {
		total += n
	}
	widths := natural
	if total > w {
		widths = make([]float32, ncol)
		for i, n := range natural {
			widths[i] = n * w / total
		}
		total = w
	}

	l := &Layout{}
	y := float32(0)
	var baseline float32
	for r := range rows {
		type cell struct {
			items []inlineItem
			lines []line
		}
		cells := make([]cell, ncol)
		rowH := float32(0)
		for ci := 0; ci < ncol; ci++ {
			items := cellItems(r, ci)
			lines, h := c.layoutInline(items, widths[ci]-2*padX, em, 1.5, false)
			cells[ci] = cell{items, lines}
			rowH = max(rowH, h+2*padY)
		}
		if r > 0 && r%2 == 0 {
			l.Prims = append(l.Prims, Prim{Kind: PRect, Y: y, W: total, H: rowH, Color: c.Theme.TableAltBg})
		}
		x := float32(0)
		for ci, cl := range cells {
			c.emitLines(l, cl.items, cl.lines, x+padX, y+padY, widths[ci]-2*padX, t.Align[ci], false)
			x += widths[ci]
		}
		if r == 0 {
			baseline = padY + firstBaseline(cells[0].lines, em)
		}
		// Horizontal rule above the row.
		l.Prims = append(l.Prims, Prim{Kind: PRect, Y: y, W: total, H: border, Color: c.Theme.Border})
		y += rowH
		if r < len(rows)-1 {
			l.Breaks = append(l.Breaks, y)
		}
	}
	l.Prims = append(l.Prims, Prim{Kind: PRect, Y: y, W: total, H: border, Color: c.Theme.Border})
	x := float32(0)
	for ci := 0; ci <= ncol; ci++ {
		l.Prims = append(l.Prims, Prim{Kind: PRect, X: min(x, total-border), W: border, H: y + border, Color: c.Theme.Border})
		if ci < ncol {
			x += widths[ci]
		}
	}
	l.Height = y + border
	return l, baseline
}

func firstBaseline(lines []line, size float32) float32 {
	if len(lines) == 0 {
		return size
	}
	return lines[0].baseline
}

// mathScale enlarges maths slightly so STIX Two matches the Go text font.
const mathScale = 1.08

func (c *Ctx) mathBlock(b *mdmodel.Block, w float32, col color.NRGBA) (*Layout, float32) {
	src := ""
	if len(b.Spans) > 0 {
		src = b.Spans[0].Text
	}
	box := tex.Layout(src, true)
	size := c.Em * mathScale
	if box.W*size > w && box.W > 0 {
		size = w / box.W // shrink formulas wider than the page
	}
	pad := 0.25 * c.Em
	x := (w - box.W*size) / 2
	switch b.Align {
	case mdmodel.AlignLeft:
		x = 0
	case mdmodel.AlignRight:
		x = w - box.W*size
	}
	baseline := pad + box.H*size
	l := &Layout{Height: baseline + box.D*size + pad}
	l.Prims = c.mathPrims(l.Prims, box, size, x, baseline, col)
	return l, baseline
}

// mathPrims converts a typeset formula into primitives, with its origin at
// (x, baseline) and em size size.
func (c *Ctx) mathPrims(out []Prim, box *tex.Box, size, x, baseline float32, col color.NRGBA) []Prim {
	for _, it := range box.Items {
		ic := col
		if it.Error {
			ic = c.Theme.Error
		}
		switch it.Kind {
		case tex.ItemGlyph:
			out = append(out, Prim{Kind: PGlyph, X: x + it.X*size, Y: baseline + it.Y*size, Size: it.Size * size,
				Face: it.Face, Glyph: it.Glyph, XScale: it.XScale, YScale: it.YScale, Color: ic})
		case tex.ItemRule:
			out = append(out, Prim{Kind: PRect, X: x + it.X*size, Y: baseline + it.Y*size,
				W: it.W * size, H: max(it.H*size, 0.5), Color: ic})
		case tex.ItemPoly:
			pts := make([][2]float32, len(it.Points))
			for i, p := range it.Points {
				pts[i] = [2]float32{x + p[0]*size, baseline + p[1]*size}
			}
			out = append(out, Prim{Kind: PPoly, Points: pts, Color: ic})
		}
	}
	return out
}
