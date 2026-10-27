// Package pdfout exports a parsed document to PDF.
package pdfout

import (
	"io"
	"path/filepath"
	"sort"
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

	"mdedit/internal/mdmodel"
	"mdedit/internal/render"
	"mdedit/internal/tex"
)

// Options control the page setup.
type Options struct {
	PageSize string  // "A4" or "Letter"
	FontSize float32 // base font size in points
	Margin   float32 // page margin in points
	BaseDir  string  // directory relative image paths are resolved against
	Title    string
}

func (o *Options) defaults() {
	if o.PageSize == "" {
		o.PageSize = "A4"
	}
	if o.FontSize == 0 {
		o.FontSize = 10.5
	}
	if o.Margin == 0 {
		o.Margin = 56
	}
}

// Font metrics of the Go fonts (hhea ascender/descender over units per em).
const ascent, descent = 0.905, 0.212

// Write renders doc as PDF to w.
func Write(w io.Writer, doc *mdmodel.Document, opt Options) error {
	opt.defaults()
	pdf := fpdf.New("P", "pt", opt.PageSize, "")
	pdf.SetMargins(float64(opt.Margin), float64(opt.Margin), float64(opt.Margin))
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetTitle(opt.Title, true)
	pdf.SetCreator("mdedit", true)
	registerFonts(pdf)

	pw, ph := pdf.GetPageSize()
	m := float64(opt.Margin)
	e := &exporter{
		pdf:    pdf,
		left:   float32(m),
		top:    float32(m),
		bottom: float32(ph - m),
		links:  map[string]int{},
		base:   opt.BaseDir,
	}
	ctx := &render.Ctx{
		Em:             opt.FontSize,
		Theme:          &render.Light,
		Measure:        e.measure,
		MaxImageHeight: float32(ph - 2*m),
		ImageSize: func(src string) (int, int, bool) {
			p, ok := render.ResolveImage(opt.BaseDir, src)
			if !ok || !supportedImage(p) {
				return 0, 0, false
			}
			return render.ImageFileSize(p)
		},
	}
	width := float32(pw - 2*m)

	// Internal links need their targets registered up front.
	for _, b := range doc.Blocks {
		if b.Kind == mdmodel.Heading && b.ID != "" {
			e.links[b.ID] = pdf.AddLink()
		}
	}

	pdf.AddPage()
	e.y = e.top
	for i := range doc.Blocks {
		b := &doc.Blocks[i]
		var prev *mdmodel.Block
		if i > 0 {
			prev = &doc.Blocks[i-1]
		}
		l := ctx.Block(b, prev, width)
		gap := b.Gap * opt.FontSize
		if e.y == e.top {
			// No space above a block at the top of a page.
			e.y -= gap
		}
		// Keep headings with the first lines of what follows them.
		if b.Kind == mdmodel.Heading && e.y > e.top && e.y+l.Height+e.keepNext(ctx, doc, i, width) > e.bottom {
			e.newPage()
			e.y -= gap
		}
		if b.Kind == mdmodel.Heading && b.ID != "" {
			pdf.SetLink(e.links[b.ID], float64(e.y+gap), -1)
		}
		e.place(l, gap)
		if err := pdf.Error(); err != nil {
			return err
		}
	}
	return pdf.Output(w)
}

// keepNext returns the height that must fit below heading i: the following
// headings plus the first two lines of the block after them.
func (e *exporter) keepNext(ctx *render.Ctx, doc *mdmodel.Document, i int, width float32) float32 {
	var h float32
	for j := i + 1; j < len(doc.Blocks); j++ {
		l := ctx.Block(&doc.Blocks[j], &doc.Blocks[j-1], width)
		if doc.Blocks[j].Kind == mdmodel.Heading {
			h += l.Height
			continue
		}
		if len(l.Breaks) > 1 {
			return h + l.Breaks[1]
		}
		return h + l.Height
	}
	return h
}

type exporter struct {
	pdf               *fpdf.Fpdf
	left, top, bottom float32
	y                 float32
	links             map[string]int
	base              string
}

func (e *exporter) newPage() {
	e.pdf.AddPage()
	e.y = e.top
}

// place draws l at the current position, splitting it over pages at its
// break points when it doesn't fit.
func (e *exporter) place(l *render.Layout, gap float32) {
	start := float32(0)
	for {
		avail := e.bottom - e.y
		if l.Height-start <= avail {
			e.draw(l, start, l.Height, e.y-start)
			e.y += l.Height - start
			return
		}
		// Find the last break that fits on this page.
		limit := start + avail
		i := sort.Search(len(l.Breaks), func(i int) bool { return l.Breaks[i] > limit })
		cut := float32(-1)
		if i > 0 && l.Breaks[i-1] > start {
			cut = l.Breaks[i-1]
		}
		if cut < 0 {
			if e.y > e.top {
				// Nothing fits here; start the block on a fresh page, without the gap.
				e.newPage()
				if start == 0 {
					start = gap
				}
				continue
			}
			// A single unbreakable piece taller than a page.
			cut = limit
		}
		e.draw(l, start, cut, e.y-start)
		e.newPage()
		start = cut
	}
}

// draw paints the part of l between y0 and y1, offset vertically by dy.
func (e *exporter) draw(l *render.Layout, y0, y1, dy float32) {
	pdf := e.pdf
	for _, p := range l.Prims {
		switch p.Kind {
		case render.PText:
			if p.Y <= y0 || p.Y > y1 {
				continue
			}
			txt := strings.TrimRight(p.Text, " ")
			if txt == "" {
				continue
			}
			e.setFont(p.Font, p.Size)
			pdf.SetTextColor(int(p.Color.R), int(p.Color.G), int(p.Color.B))
			pdf.Text(float64(e.left+p.X), float64(p.Y+dy), txt)
		case render.PRect, render.PEllipse:
			top, bot := max(p.Y, y0), min(p.Y+p.H, y1)
			if bot <= top {
				continue
			}
			x, y, w, h := float64(e.left+p.X), float64(top+dy), float64(p.W), float64(bot-top)
			style := "F"
			if p.Stroke > 0 {
				style = "D"
				pdf.SetLineWidth(float64(p.Stroke))
				pdf.SetDrawColor(int(p.Color.R), int(p.Color.G), int(p.Color.B))
			}
			pdf.SetFillColor(int(p.Color.R), int(p.Color.G), int(p.Color.B))
			switch {
			case p.Kind == render.PEllipse:
				pdf.Ellipse(x+w/2, y+h/2, w/2, h/2, 0, style)
			case p.Radius > 0:
				// Keep rounded corners only on edges that weren't cut.
				corners := ""
				if top == p.Y {
					corners += "12"
				}
				if bot == p.Y+p.H {
					corners += "34"
				}
				pdf.RoundedRect(x, y, w, h, float64(min(p.Radius, (bot-top)/2)), corners, style)
			default:
				pdf.Rect(x, y, w, h, style)
			}
		case render.PLine:
			if min(p.Y, p.H) < y0 || max(p.Y, p.H) > y1 {
				continue
			}
			pdf.SetLineWidth(float64(p.Stroke))
			pdf.SetDrawColor(int(p.Color.R), int(p.Color.G), int(p.Color.B))
			pdf.SetLineCapStyle("round")
			pdf.Line(float64(e.left+p.X), float64(p.Y+dy), float64(e.left+p.W), float64(p.H+dy))
		case render.PGlyph:
			if p.Y <= y0 || p.Y > y1 {
				continue
			}
			segs := tex.Outline(p.Face, p.Glyph)
			if len(segs) == 0 {
				continue
			}
			sx, sy := p.Size*p.XScale, p.Size*p.YScale
			pt := func(q [2]float32) (float64, float64) {
				return float64(e.left + p.X + q[0]*sx), float64(p.Y + dy + q[1]*sy)
			}
			pdf.SetFillColor(int(p.Color.R), int(p.Color.G), int(p.Color.B))
			for i, s := range segs {
				switch s.Op {
				case tex.MoveTo:
					if i > 0 {
						pdf.ClosePath()
					}
					pdf.MoveTo(pt(s.P[0]))
				case tex.LineTo:
					pdf.LineTo(pt(s.P[0]))
				case tex.QuadTo:
					cx, cy := pt(s.P[0])
					x, y := pt(s.P[1])
					pdf.CurveTo(cx, cy, x, y)
				case tex.CubeTo:
					c1x, c1y := pt(s.P[0])
					c2x, c2y := pt(s.P[1])
					x, y := pt(s.P[2])
					pdf.CurveBezierCubicTo(c1x, c1y, c2x, c2y, x, y)
				}
			}
			pdf.ClosePath()
			pdf.DrawPath("F")
		case render.PPoly:
			cy := (p.Top() + p.Points[len(p.Points)/2][1]) / 2
			if cy <= y0 || cy > y1 {
				continue
			}
			pdf.SetFillColor(int(p.Color.R), int(p.Color.G), int(p.Color.B))
			for i, q := range p.Points {
				x, y := float64(e.left+q[0]), float64(q[1]+dy)
				if i == 0 {
					pdf.MoveTo(x, y)
				} else {
					pdf.LineTo(x, y)
				}
			}
			pdf.ClosePath()
			pdf.DrawPath("F")
		case render.PImage:
			if p.Y < y0 || p.Y >= y1 {
				continue
			}
			path, ok := render.ResolveImage(e.base, p.URL)
			if !ok {
				continue
			}
			pdf.ImageOptions(path, float64(e.left+p.X), float64(p.Y+dy), float64(p.W), float64(p.H),
				false, fpdf.ImageOptions{ImageType: imageType(path)}, 0, "")
		case render.PLink:
			if p.Y < y0 || p.Y >= y1 {
				continue
			}
			x, y, w, h := float64(e.left+p.X), float64(p.Y+dy), float64(p.W), float64(p.H)
			if strings.HasPrefix(p.URL, "#") {
				if id, ok := e.links[p.URL[1:]]; ok {
					pdf.Link(x, y, w, h, id)
				}
			} else {
				pdf.LinkString(x, y, w, h, p.URL)
			}
		}
	}
}

func (e *exporter) setFont(f render.FontSpec, size float32) {
	family := "go"
	if f.Mono {
		family = "gomono"
	}
	style := ""
	if f.Bold {
		style += "B"
	}
	if f.Italic {
		style += "I"
	}
	e.pdf.SetFont(family, style, float64(size))
}

func (e *exporter) measure(f render.FontSpec, size float32, s string) render.Metrics {
	e.setFont(f, size)
	return render.Metrics{
		Width:   float32(e.pdf.GetStringWidth(s)),
		Ascent:  ascent * size,
		Descent: descent * size,
	}
}

func registerFonts(pdf *fpdf.Fpdf) {
	fonts := []struct {
		family, style string
		ttf           []byte
	}{
		{"go", "", goregular.TTF},
		{"go", "B", gobold.TTF},
		{"go", "I", goitalic.TTF},
		{"go", "BI", gobolditalic.TTF},
		{"gomono", "", gomono.TTF},
		{"gomono", "B", gomonobold.TTF},
		{"gomono", "I", gomonoitalic.TTF},
		{"gomono", "BI", gomonobolditalic.TTF},
	}
	for _, f := range fonts {
		pdf.AddUTF8FontFromBytes(f.family, f.style, f.ttf)
	}
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
