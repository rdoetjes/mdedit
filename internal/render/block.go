package render

import (
	"image/color"

	"mdedit/internal/mdmodel"
)

// frame is the space a block's containers (quotes, list items) take up.
type frame struct {
	x0        float32 // left edge of the content
	bars      []quoteBar
	markers   []itemMarker
	textColor color.NRGBA
}

type quoteBar struct {
	x     float32
	joins bool // continues the bar of the previous block
}

type itemMarker struct {
	x float32
	d mdmodel.Decor
}

// Block lays out one block at the given width. prev is the preceding block,
// or nil, and decides whether block quote bars join up.
func (c *Ctx) Block(b, prev *mdmodel.Block, width float32) *Layout {
	gap := b.Gap * c.Em
	f := c.frame(b, prev)
	content, baseline := c.content(b, max(width-f.x0, 4*c.Em), f.textColor)
	out := offsetLayout(content, f.x0, gap)
	out.Height = gap + content.Height
	c.addBars(out, f.bars, gap)
	for _, m := range f.markers {
		out.Prims = append(out.Prims, c.marker(m.x, baseline+gap, m.d, f.textColor)...)
	}
	return out
}

// frame works out the indentation, quote bars and list markers of b.
func (c *Ctx) frame(b, prev *mdmodel.Block) frame {
	f := frame{textColor: c.Theme.Text}
	for i, d := range b.Decor {
		switch d.Kind {
		case mdmodel.Quote:
			joins := prev != nil && i < len(prev.Decor) && prev.Decor[i].ID == d.ID
			f.bars = append(f.bars, quoteBar{x: f.x0, joins: joins})
			f.x0 += 0.25*c.Em + c.Em
			f.textColor = c.Theme.Muted
		case mdmodel.Item:
			if d.ShowMarker {
				f.markers = append(f.markers, itemMarker{x: f.x0, d: d})
			}
			f.x0 += 2 * c.Em
		}
	}
	return f
}

// content lays out the block itself, returning the baseline of its first
// line.
func (c *Ctx) content(b *mdmodel.Block, w float32, col color.NRGBA) (*Layout, float32) {
	switch b.Kind {
	case mdmodel.Heading:
		return c.heading(b, w, col)
	case mdmodel.Code, mdmodel.HTML:
		return c.code(b, w)
	case mdmodel.Rule:
		return c.rule(w)
	case mdmodel.Table:
		return c.table(b.Table, w, col)
	case mdmodel.Image:
		return c.image(b, w, col)
	case mdmodel.Math:
		return c.mathBlock(b, w, col)
	}
	return c.paragraph(b.Spans, w, col, b.Align)
}

func (c *Ctx) rule(w float32) (*Layout, float32) {
	h := 0.25 * c.Em
	return &Layout{Height: h, Prims: []Prim{{Kind: PRect, W: w, H: h, Color: c.Theme.Border}}}, h
}

// offsetLayout returns a copy of l's primitives and breaks moved by (dx, dy).
func offsetLayout(l *Layout, dx, dy float32) *Layout {
	out := &Layout{Prims: make([]Prim, len(l.Prims))}
	for i, p := range l.Prims {
		out.Prims[i] = offsetPrim(p, dx, dy)
	}
	for _, br := range l.Breaks {
		out.Breaks = append(out.Breaks, br+dy)
	}
	return out
}

func offsetPrim(p Prim, dx, dy float32) Prim {
	p.X += dx
	p.Y += dy
	switch p.Kind {
	case PLine:
		p.W += dx
		p.H += dy
	case PPoly:
		p.Points = offsetPoints(p.Points, dx, dy)
	}
	return p
}

func offsetPoints(pts [][2]float32, dx, dy float32) [][2]float32 {
	out := make([][2]float32, len(pts))
	for i, q := range pts {
		out[i] = [2]float32{q[0] + dx, q[1] + dy}
	}
	return out
}

// addBars draws the block quote bars, joining them to the previous block's
// across the gap where the quote continues.
func (c *Ctx) addBars(out *Layout, bars []quoteBar, gap float32) {
	for _, br := range bars {
		top := gap
		if br.joins {
			top = 0
		}
		out.Prims = append(out.Prims, Prim{Kind: PRect, X: br.x, Y: top, W: 0.25 * c.Em, H: out.Height - top, Color: c.Theme.Border})
	}
}
