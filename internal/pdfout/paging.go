package pdfout

import "mdedit/internal/render"

// place draws l at the current position, splitting it over pages at its
// break points when it doesn't fit.
func (e *exporter) place(l *render.Layout, gap float32) {
	start := float32(0)
	for l.Height-start > e.bottom-e.y {
		start = e.fillPage(l, start, gap)
	}
	e.draw(l, segment{y0: start, y1: l.Height, dy: e.y - start})
	e.y += l.Height - start
}

// fillPage draws as much of l from start as fits on this page, starts a new
// page, and returns where to continue.
func (e *exporter) fillPage(l *render.Layout, start, gap float32) float32 {
	limit := start + (e.bottom - e.y)
	cut := lastBreak(l.Breaks, start, limit)
	if cut < 0 && e.y > e.top {
		// Nothing fits here: start the block on a fresh page, without the gap.
		e.newPage()
		if start == 0 {
			return gap
		}
		return start
	}
	if cut < 0 {
		cut = limit // a single unbreakable piece taller than a page
	}
	e.draw(l, segment{y0: start, y1: cut, dy: e.y - start})
	e.newPage()
	return cut
}

// lastBreak returns the last break after start and at most limit, or -1.
func lastBreak(breaks []float32, start, limit float32) float32 {
	cut := float32(-1)
	for _, b := range breaks {
		if b > limit {
			break
		}
		if b > start {
			cut = b
		}
	}
	return cut
}

// segment is the part of a layout between y0 and y1 that goes on the
// current page, shifted down by dy.
type segment struct {
	y0, y1, dy float32
}

// hasBaseline reports whether text with baseline y belongs to the segment.
func (s segment) hasBaseline(y float32) bool { return y > s.y0 && y <= s.y1 }

// hasTop reports whether an element starting at y belongs to the segment.
func (s segment) hasTop(y float32) bool { return y >= s.y0 && y < s.y1 }
