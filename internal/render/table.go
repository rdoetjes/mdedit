package render

import (
	"image/color"

	"mdedit/internal/mdmodel"
)

// tableLayout lays out a table: header row, body rows and grid lines.
type tableLayout struct {
	c          *Ctx
	t          *mdmodel.TableData
	rows       [][][]mdmodel.Span // header first
	col        color.NRGBA
	padX, padY float32
	border     float32
	widths     []float32
	total      float32
	l          *Layout
	y          float32
	baseline   float32
}

func (c *Ctx) table(t *mdmodel.TableData, w float32, col color.NRGBA) (*Layout, float32) {
	tl := &tableLayout{
		c: c, t: t, col: col, l: &Layout{},
		rows:   append([][][]mdmodel.Span{t.Header}, t.Rows...),
		padX:   13 * c.Em / 16,
		padY:   6 * c.Em / 16,
		border: max(1, c.Em/16),
	}
	tl.columnWidths(w)
	for r := range tl.rows {
		tl.row(r)
	}
	tl.grid()
	tl.l.Height = tl.y + tl.border
	return tl.l, tl.baseline
}

func (tl *tableLayout) ncol() int { return len(tl.t.Align) }

// cellItems returns the items of a cell; header cells are bold.
func (tl *tableLayout) cellItems(r, ci int) []inlineItem {
	if ci >= len(tl.rows[r]) {
		return nil
	}
	spans := tl.rows[r][ci]
	if r == 0 {
		spans = boldSpans(spans)
	}
	return tl.c.items(spans, tl.c.Em, tl.col, false)
}

func boldSpans(in []mdmodel.Span) []mdmodel.Span {
	out := append([]mdmodel.Span(nil), in...)
	for i := range out {
		out[i].Bold = true
	}
	return out
}

// columnWidths uses the natural column widths, shrunk proportionally if the
// table is wider than w.
func (tl *tableLayout) columnWidths(w float32) {
	tl.widths = tl.naturalWidths()
	tl.total = 0
	for _, n := range tl.widths {
		tl.total += n
	}
	if tl.total <= w {
		return
	}
	for i := range tl.widths {
		tl.widths[i] *= w / tl.total
	}
	tl.total = w
}

func (tl *tableLayout) naturalWidths() []float32 {
	natural := make([]float32, tl.ncol())
	for r := range tl.rows {
		for ci := range natural {
			natural[ci] = max(natural[ci], tl.naturalWidth(r, ci))
		}
	}
	return natural
}

func (tl *tableLayout) naturalWidth(r, ci int) float32 {
	lines, _ := tl.c.layoutInline(tl.cellItems(r, ci), 1e9, tl.c.Em, 1.5, false)
	w := 2*tl.padX + tl.c.Em
	for _, ln := range lines {
		w = max(w, ln.width+2*tl.padX)
	}
	return w
}

type tableCell struct {
	items []inlineItem
	lines []line
}

// row lays out row r below the previous rows.
func (tl *tableLayout) row(r int) {
	cells, rowH := tl.layoutCells(r)
	if r > 0 && r%2 == 0 {
		tl.add(Prim{Kind: PRect, Y: tl.y, W: tl.total, H: rowH, Color: tl.c.Theme.TableAltBg})
	}
	x := float32(0)
	for ci, cl := range cells {
		tl.c.emitLines(tl.l, cl.items, cl.lines, x+tl.padX, tl.y+tl.padY, tl.widths[ci]-2*tl.padX, tl.t.Align[ci], false)
		x += tl.widths[ci]
	}
	if r == 0 {
		tl.baseline = tl.padY + firstBaseline(cells[0].lines, tl.c.Em)
	}
	tl.hline(tl.y)
	tl.y += rowH
	if r < len(tl.rows)-1 {
		tl.l.Breaks = append(tl.l.Breaks, tl.y)
	}
}

func (tl *tableLayout) layoutCells(r int) ([]tableCell, float32) {
	cells := make([]tableCell, tl.ncol())
	rowH := float32(0)
	for ci := range cells {
		items := tl.cellItems(r, ci)
		lines, h := tl.c.layoutInline(items, tl.widths[ci]-2*tl.padX, tl.c.Em, 1.5, false)
		cells[ci] = tableCell{items, lines}
		rowH = max(rowH, h+2*tl.padY)
	}
	return cells, rowH
}

func (tl *tableLayout) add(p Prim) { tl.l.Prims = append(tl.l.Prims, p) }

func (tl *tableLayout) hline(y float32) {
	tl.add(Prim{Kind: PRect, Y: y, W: tl.total, H: tl.border, Color: tl.c.Theme.Border})
}

// grid draws the bottom rule and the vertical lines.
func (tl *tableLayout) grid() {
	tl.hline(tl.y)
	x := float32(0)
	for ci := 0; ci <= tl.ncol(); ci++ {
		tl.add(Prim{Kind: PRect, X: min(x, tl.total-tl.border), W: tl.border, H: tl.y + tl.border, Color: tl.c.Theme.Border})
		if ci < tl.ncol() {
			x += tl.widths[ci]
		}
	}
}
