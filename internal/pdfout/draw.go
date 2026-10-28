package pdfout

import (
	"image/color"
	"strings"

	"github.com/go-pdf/fpdf"

	"mdedit/internal/render"
)

// draw paints the primitives of l that fall in seg.
func (e *exporter) draw(l *render.Layout, seg segment) {
	for i := range l.Prims {
		e.drawPrim(&l.Prims[i], seg)
	}
}

func (e *exporter) drawPrim(p *render.Prim, seg segment) {
	switch p.Kind {
	case render.PText:
		e.drawTextPrim(p, seg)
	case render.PRect, render.PEllipse:
		e.drawShape(p, seg)
	case render.PLine:
		e.drawLine(p, seg)
	case render.PGlyph:
		if seg.hasBaseline(p.Y) {
			e.fillGlyph(p.Face, p.Glyph, e.left+p.X, p.Y+seg.dy, p.Size*p.XScale, p.Size*p.YScale, p.Color)
		}
	case render.PPoly:
		e.drawPoly(p, seg)
	case render.PImage:
		e.drawImage(p, seg)
	case render.PLink:
		e.drawLink(p, seg)
	}
}

func (e *exporter) drawTextPrim(p *render.Prim, seg segment) {
	txt := strings.TrimRight(p.Text, " ")
	if txt != "" && seg.hasBaseline(p.Y) {
		e.drawText(p.Font, p.Size, txt, e.left+p.X, p.Y+seg.dy, p.Color)
	}
}

func (e *exporter) setFill(c color.NRGBA) { e.pdf.SetFillColor(int(c.R), int(c.G), int(c.B)) }
func (e *exporter) setDraw(c color.NRGBA) { e.pdf.SetDrawColor(int(c.R), int(c.G), int(c.B)) }

// drawShape draws a rectangle or ellipse, clipped to the segment.
func (e *exporter) drawShape(p *render.Prim, seg segment) {
	top, bot := max(p.Y, seg.y0), min(p.Y+p.H, seg.y1)
	if bot <= top {
		return
	}
	style := e.shapeStyle(p)
	x, y, w, h := float64(e.left+p.X), float64(top+seg.dy), float64(p.W), float64(bot-top)
	switch {
	case p.Kind == render.PEllipse:
		e.pdf.Ellipse(x+w/2, y+h/2, w/2, h/2, 0, style)
	case p.Radius > 0:
		r := float64(min(p.Radius, (bot-top)/2))
		e.pdf.RoundedRect(x, y, w, h, r, keptCorners(p, top, bot), style)
	default:
		e.pdf.Rect(x, y, w, h, style)
	}
}

// shapeStyle sets the colours for a filled or outlined shape and returns
// the fpdf style.
func (e *exporter) shapeStyle(p *render.Prim) string {
	e.setFill(p.Color)
	if p.Stroke <= 0 {
		return "F"
	}
	e.pdf.SetLineWidth(float64(p.Stroke))
	e.setDraw(p.Color)
	return "D"
}

// keptCorners lists the rounded corners on edges that weren't cut by a page
// break.
func keptCorners(p *render.Prim, top, bot float32) string {
	corners := ""
	if top == p.Y {
		corners += "12"
	}
	if bot == p.Y+p.H {
		corners += "34"
	}
	return corners
}

// drawLine strokes the line from (X, Y) to (W, H).
func (e *exporter) drawLine(p *render.Prim, seg segment) {
	if min(p.Y, p.H) < seg.y0 || max(p.Y, p.H) > seg.y1 {
		return
	}
	e.pdf.SetLineWidth(float64(p.Stroke))
	e.setDraw(p.Color)
	e.pdf.SetLineCapStyle("round")
	e.pdf.Line(float64(e.left+p.X), float64(p.Y+seg.dy), float64(e.left+p.W), float64(p.H+seg.dy))
}

func (e *exporter) drawPoly(p *render.Prim, seg segment) {
	if !seg.hasBaseline((p.Top() + p.Points[len(p.Points)/2][1]) / 2) {
		return
	}
	e.setFill(p.Color)
	e.pdf.MoveTo(float64(e.left+p.Points[0][0]), float64(p.Points[0][1]+seg.dy))
	for _, q := range p.Points[1:] {
		e.pdf.LineTo(float64(e.left+q[0]), float64(q[1]+seg.dy))
	}
	e.pdf.ClosePath()
	e.pdf.DrawPath("F")
}

func (e *exporter) drawImage(p *render.Prim, seg segment) {
	if !seg.hasTop(p.Y) {
		return
	}
	path, ok := render.ResolveImage(e.base, p.URL)
	if !ok {
		return
	}
	e.pdf.ImageOptions(path, float64(e.left+p.X), float64(p.Y+seg.dy), float64(p.W), float64(p.H),
		false, fpdf.ImageOptions{ImageType: imageType(path)}, 0, "")
}

// drawLink adds a link area: to a heading for "#anchor", else to a URL.
func (e *exporter) drawLink(p *render.Prim, seg segment) {
	if !seg.hasTop(p.Y) {
		return
	}
	x, y, w, h := float64(e.left+p.X), float64(p.Y+seg.dy), float64(p.W), float64(p.H)
	anchor, internal := strings.CutPrefix(p.URL, "#")
	if !internal {
		e.pdf.LinkString(x, y, w, h, p.URL)
		return
	}
	if id, ok := e.links[anchor]; ok {
		e.pdf.Link(x, y, w, h, id)
	}
}
