package ui

import (
	"image"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"mdedit/internal/render"
	"mdedit/internal/tex"
)

// paint draws a block layout with its left edge at ox.
func (p *Preview) paint(gtx layout.Context, l *render.Layout, ox float32) {
	for i := range l.Prims {
		p.paintPrim(gtx, &l.Prims[i], ox)
	}
}

func (p *Preview) paintPrim(gtx layout.Context, pr *render.Prim, ox float32) {
	x := pr.X + ox
	switch pr.Kind {
	case render.PRect:
		paintRect(gtx, pr, x)
	case render.PEllipse:
		paintEllipse(gtx, pr, x)
	case render.PLine:
		paintLine(gtx, pr, ox)
	case render.PText:
		p.text.draw(gtx, pr.Font, pr.Size, pr.Text, x, pr.Y, pr.Color)
	case render.PImage:
		p.paintImage(gtx, pr, x)
	case render.PGlyph:
		paintGlyph(gtx, pr, x)
	case render.PPoly:
		paintPoly(gtx, pr, ox)
	case render.PLink:
		linkCursor(gtx, pr, x)
	}
}

func primRect(pr *render.Prim, x float32) image.Rectangle {
	return image.Rect(rnd(x), rnd(pr.Y), rnd(x+pr.W), rnd(pr.Y+pr.H))
}

func paintRect(gtx layout.Context, pr *render.Prim, x float32) {
	rr := clip.UniformRRect(primRect(pr, x), rnd(pr.Radius))
	if pr.Stroke > 0 {
		paint.FillShape(gtx.Ops, pr.Color, clip.Stroke{Path: rr.Path(gtx.Ops), Width: pr.Stroke}.Op())
		return
	}
	paint.FillShape(gtx.Ops, pr.Color, rr.Op(gtx.Ops))
}

func paintEllipse(gtx layout.Context, pr *render.Prim, x float32) {
	r := primRect(pr, x)
	e := clip.Ellipse{Min: r.Min, Max: r.Max}
	if pr.Stroke > 0 {
		paint.FillShape(gtx.Ops, pr.Color, clip.Stroke{Path: e.Path(gtx.Ops), Width: pr.Stroke}.Op())
		return
	}
	paint.FillShape(gtx.Ops, pr.Color, e.Op(gtx.Ops))
}

// paintLine strokes the line from (X, Y) to (W, H).
func paintLine(gtx layout.Context, pr *render.Prim, ox float32) {
	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(f32.Pt(pr.X+ox, pr.Y))
	path.LineTo(f32.Pt(pr.W+ox, pr.H))
	paint.FillShape(gtx.Ops, pr.Color, clip.Stroke{Path: path.End(), Width: pr.Stroke}.Op())
}

func (p *Preview) paintImage(gtx layout.Context, pr *render.Prim, x float32) {
	img := p.image(pr.URL)
	if img == nil {
		return
	}
	scale := f32.Pt(pr.W/float32(img.size.X), pr.H/float32(img.size.Y))
	at := f32.Pt(float32(rnd(x)), float32(rnd(pr.Y)))
	st := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, scale).Offset(at)).Push(gtx.Ops)
	img.op.Add(gtx.Ops)
	cl := clip.Rect{Max: img.size}.Push(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	cl.Pop()
	st.Pop()
}

// glyphTransform maps glyph outline coordinates (em, y down) to the page.
type glyphTransform struct {
	x, y, sx, sy float32
}

func (t glyphTransform) pt(q [2]float32) f32.Point {
	return f32.Pt(t.x+q[0]*t.sx, t.y+q[1]*t.sy)
}

func paintGlyph(gtx layout.Context, pr *render.Prim, x float32) {
	segs := tex.Outline(pr.Face, pr.Glyph)
	if len(segs) == 0 {
		return
	}
	t := glyphTransform{x: x, y: pr.Y, sx: pr.Size * pr.XScale, sy: pr.Size * pr.YScale}
	var path clip.Path
	path.Begin(gtx.Ops)
	for i, s := range segs {
		addSegment(&path, s, t, i > 0)
	}
	path.Close()
	paint.FillShape(gtx.Ops, pr.Color, clip.Outline{Path: path.End()}.Op())
}

// addSegment appends one outline segment; a MoveTo after the first segment
// closes the previous contour.
func addSegment(path *clip.Path, s tex.Seg, t glyphTransform, closePrev bool) {
	switch s.Op {
	case tex.MoveTo:
		if closePrev {
			path.Close()
		}
		path.MoveTo(t.pt(s.P[0]))
	case tex.LineTo:
		path.LineTo(t.pt(s.P[0]))
	case tex.QuadTo:
		path.QuadTo(t.pt(s.P[0]), t.pt(s.P[1]))
	case tex.CubeTo:
		path.CubeTo(t.pt(s.P[0]), t.pt(s.P[1]), t.pt(s.P[2]))
	}
}

func paintPoly(gtx layout.Context, pr *render.Prim, ox float32) {
	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(f32.Pt(pr.Points[0][0]+ox, pr.Points[0][1]))
	for _, q := range pr.Points[1:] {
		path.LineTo(f32.Pt(q[0]+ox, q[1]))
	}
	path.Close()
	paint.FillShape(gtx.Ops, pr.Color, clip.Outline{Path: path.End()}.Op())
}

// linkCursor shows the hand cursor over a link.
func linkCursor(gtx layout.Context, pr *render.Prim, x float32) {
	area := clip.Rect(primRect(pr, x)).Push(gtx.Ops)
	pointer.CursorPointer.Add(gtx.Ops)
	area.Pop()
}
