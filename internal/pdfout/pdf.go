// Package pdfout exports a parsed document to PDF.
package pdfout

import (
	"io"

	"github.com/go-pdf/fpdf"

	"mdedit/internal/mdmodel"
	"mdedit/internal/render"
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
	e := newExporter(opt)
	e.registerLinks(doc)
	e.pdf.AddPage()
	e.y = e.top
	for i := range doc.Blocks {
		e.placeBlock(doc, i)
		if err := e.pdf.Error(); err != nil {
			return err
		}
	}
	return e.pdf.Output(w)
}

type exporter struct {
	pdf               *fpdf.Fpdf
	ctx               *render.Ctx
	left, top, bottom float32
	width             float32
	y                 float32
	links             map[string]int
	coverage          map[coverKey]bool
	base              string
}

func newExporter(opt Options) *exporter {
	pdf := fpdf.New("P", "pt", opt.PageSize, "")
	m := float64(opt.Margin)
	pdf.SetMargins(m, m, m)
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetTitle(opt.Title, true)
	pdf.SetCreator("mdedit", true)
	registerFonts(pdf)
	pw, ph := pdf.GetPageSize()
	e := &exporter{
		pdf:      pdf,
		left:     opt.Margin,
		top:      opt.Margin,
		bottom:   float32(ph - m),
		width:    float32(pw - 2*m),
		links:    map[string]int{},
		coverage: map[coverKey]bool{},
		base:     opt.BaseDir,
	}
	e.ctx = &render.Ctx{Em: opt.FontSize, Theme: &render.Light, Measure: e.measure,
		MaxImageHeight: float32(ph - 2*m), ImageSize: e.imageSize}
	return e
}

// imageSize reports the size of an image the PDF can embed.
func (e *exporter) imageSize(src string) (int, int, bool) {
	p, ok := render.ResolveImage(e.base, src)
	if !ok || !supportedImage(p) {
		return 0, 0, false
	}
	return render.ImageFileSize(p)
}

// registerLinks creates link targets for all headings, since internal links
// may point forward.
func (e *exporter) registerLinks(doc *mdmodel.Document) {
	for _, b := range doc.Blocks {
		if b.Kind == mdmodel.Heading && b.ID != "" {
			e.links[b.ID] = e.pdf.AddLink()
		}
	}
}

// placeBlock lays out block i and puts it on the page.
func (e *exporter) placeBlock(doc *mdmodel.Document, i int) {
	b := &doc.Blocks[i]
	var prev *mdmodel.Block
	if i > 0 {
		prev = &doc.Blocks[i-1]
	}
	l := e.ctx.Block(b, prev, e.width)
	gap := b.Gap * e.ctx.Em
	if e.y == e.top {
		e.y -= gap // no space above a block at the top of a page
	}
	if b.Kind == mdmodel.Heading && e.y > e.top && e.y+l.Height+e.keepNext(doc, i) > e.bottom {
		e.newPage() // keep the heading with what follows it
		e.y -= gap
	}
	if b.Kind == mdmodel.Heading && b.ID != "" {
		e.pdf.SetLink(e.links[b.ID], float64(e.y+gap), -1)
	}
	e.place(l, gap)
}

// keepNext returns the height that must fit below heading i: the following
// headings plus the first two lines of the block after them.
func (e *exporter) keepNext(doc *mdmodel.Document, i int) float32 {
	var h float32
	for j := i + 1; j < len(doc.Blocks); j++ {
		l := e.ctx.Block(&doc.Blocks[j], &doc.Blocks[j-1], e.width)
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

func (e *exporter) newPage() {
	e.pdf.AddPage()
	e.y = e.top
}
