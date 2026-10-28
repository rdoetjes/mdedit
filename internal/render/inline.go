package render

import (
	"image/color"
	"strings"
	"unicode"
	"unicode/utf8"

	"mdedit/internal/mdmodel"
	"mdedit/internal/tex"
)

// inlineItem is a span prepared for layout.
type inlineItem struct {
	span  mdmodel.Span
	math  *tex.Box // inline maths, drawn at size*mathScale
	font  FontSpec
	size  float32
	color color.NRGBA
}

type run struct {
	item         int
	text         string
	x, w         float32
	ascent, desc float32
}

type line struct {
	runs     []run
	top      float32
	height   float32
	baseline float32
	width    float32
}

// items converts spans to layout items. In code blocks (code=true) all text
// is already monospace at the given size; elsewhere inline code is set
// smaller than the surrounding text.
func (c *Ctx) items(spans []mdmodel.Span, size float32, col color.NRGBA, code bool) []inlineItem {
	out := make([]inlineItem, len(spans))
	for i, s := range spans {
		out[i] = c.item(s, size, col, code)
	}
	return out
}

func (c *Ctx) item(s mdmodel.Span, size float32, col color.NRGBA, code bool) inlineItem {
	it := inlineItem{span: s, font: FontSpec{Mono: s.Mono, Bold: s.Bold, Italic: s.Italic}, size: size, color: col}
	if s.Mono && !code {
		it.size = size * codeScale
	}
	if s.Math {
		it.math = tex.Layout(s.Text, s.Display)
	}
	if code {
		it.span.Text = strings.ReplaceAll(s.Text, "\t", "    ")
	}
	switch {
	case s.Color != (color.NRGBA{}):
		it.color = s.Color
	case s.Link != "":
		it.color = c.Theme.Link
	}
	return it
}

// tokens splits s into words with their trailing spaces, and lone "\n"
// tokens for hard line breaks.
func tokens(s string) []string {
	var out []string
	for _, part := range strings.SplitAfter(s, "\n") {
		text, hasBreak := strings.CutSuffix(part, "\n")
		out = append(out, words(text)...)
		if hasBreak {
			out = append(out, "\n")
		}
	}
	return out
}

// words splits text (without newlines) after each run of spaces.
func words(text string) []string {
	var out []string
	start := 0
	inSpace := false
	for i, r := range text {
		space := unicode.IsSpace(r)
		if inSpace && !space {
			out = append(out, text[start:i])
			start = i
		}
		inSpace = space
	}
	if start < len(text) {
		out = append(out, text[start:])
	}
	return out
}

// measurer measures text in one font and size.
type measurer struct {
	c    *Ctx
	font FontSpec
	size float32
}

func (m measurer) measure(s string) Metrics { return m.c.Measure(m.font, m.size, s) }

// fitRunes returns the byte length of the longest prefix of s (at least one
// rune) that fits in w.
func fitRunes(s string, w float32, m measurer) int {
	var idx []int
	for i := range s {
		if i > 0 {
			idx = append(idx, i)
		}
	}
	idx = append(idx, len(s))
	lo, hi := 0, len(idx)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if m.measure(s[:idx[mid]]).Width <= w {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return idx[lo]
}

// layoutInline breaks items into lines no wider than maxW. strut is the
// minimum font size used for line metrics, so lines containing only smaller
// text keep a regular height. With preserve set, leading spaces on lines are
// kept (code blocks); otherwise they are dropped at wrap points.
func (c *Ctx) layoutInline(items []inlineItem, maxW, strut, lineHeight float32, preserve bool) ([]line, float32) {
	lb := &lineBreaker{c: c, items: items, maxW: maxW, strut: strut, lineHeight: lineHeight, preserve: preserve}
	lb.strutM = c.Measure(FontSpec{Mono: preserve}, strut, "x")
	if lb.strutM.Ascent == 0 {
		lb.strutM.Ascent, lb.strutM.Descent = 0.9*strut, 0.22*strut
	}
	for i := range items {
		lb.addItem(i)
	}
	if len(lb.cur.runs) > 0 || len(lb.lines) == 0 {
		lb.finish()
	}
	return lb.lines, lb.y
}

// lineBreaker fills lines with runs of text.
type lineBreaker struct {
	c           *Ctx
	items       []inlineItem
	maxW, strut float32
	lineHeight  float32
	preserve    bool
	strutM      Metrics
	lines       []line
	cur         line
	x, y        float32
}

func (lb *lineBreaker) addItem(i int) {
	it := lb.items[i]
	if it.math != nil {
		lb.addFormula(i, it)
		return
	}
	m := measurer{c: lb.c, font: it.font, size: it.size}
	for _, t := range tokens(it.span.Text) {
		lb.addToken(i, t, m)
	}
}

// addFormula places inline maths as a single unbreakable box.
func (lb *lineBreaker) addFormula(i int, it inlineItem) {
	ms := it.size * mathScale
	m := Metrics{Width: it.math.W * ms, Ascent: it.math.H * ms, Descent: it.math.D * ms}
	if lb.x > 0 && lb.x+m.Width > lb.maxW {
		lb.finish()
	}
	lb.place(i, "", m)
}

// addToken places a word with its trailing spaces, wrapping first if it
// doesn't fit.
func (lb *lineBreaker) addToken(i int, t string, m measurer) {
	if t == "\n" {
		lb.finish()
		return
	}
	if lb.x > 0 && lb.x+m.measure(trimRight(t)).Width > lb.maxW {
		lb.finish()
	}
	if lb.x == 0 && !lb.preserve {
		t = strings.TrimLeftFunc(t, unicode.IsSpace)
	}
	if t == "" {
		return
	}
	t = lb.breakLongWord(i, t, m)
	lb.place(i, t, m.measure(t))
}

// breakLongWord places the parts of a word too long for a line on lines of
// their own and returns the rest.
func (lb *lineBreaker) breakLongWord(i int, t string, m measurer) string {
	word := trimRight(t)
	for m.measure(word).Width > lb.maxW && utf8.RuneCountInString(word) > 1 {
		k := fitRunes(word, lb.maxW-lb.x, m)
		lb.place(i, word[:k], m.measure(word[:k]))
		lb.finish()
		t, word = t[k:], word[k:]
	}
	return t
}

func trimRight(s string) string { return strings.TrimRightFunc(s, unicode.IsSpace) }

// place appends text to the current line, extending the last run if it
// belongs to the same item.
func (lb *lineBreaker) place(i int, t string, m Metrics) {
	if n := len(lb.cur.runs); n > 0 && lb.cur.runs[n-1].item == i {
		r := &lb.cur.runs[n-1]
		r.text += t
		r.w += m.Width
	} else {
		lb.cur.runs = append(lb.cur.runs, run{item: i, text: t, x: lb.x, w: m.Width, ascent: m.Ascent, desc: m.Descent})
	}
	lb.x += m.Width
}

// finish ends the current line, working out its height and baseline.
func (lb *lineBreaker) finish() {
	asc, desc, h := lb.lineMetrics()
	lb.cur.top = lb.y
	lb.cur.height = h
	lb.cur.baseline = lb.y + (h-(asc+desc))/2 + asc
	lb.cur.width = lb.x - lb.trailingSpace()
	lb.lines = append(lb.lines, lb.cur)
	lb.y += h
	lb.cur = line{}
	lb.x = 0
}

func (lb *lineBreaker) lineMetrics() (asc, desc, h float32) {
	asc, desc = lb.strutM.Ascent, lb.strutM.Descent
	h = lb.strut * lb.lineHeight
	for _, r := range lb.cur.runs {
		asc = max(asc, r.ascent)
		desc = max(desc, r.desc)
		h = max(h, lb.items[r.item].size*lb.lineHeight)
	}
	return asc, desc, max(h, asc+desc)
}

// trailingSpace is the width of spaces ending the line, which don't count
// for alignment.
func (lb *lineBreaker) trailingSpace() float32 {
	n := len(lb.cur.runs)
	if n == 0 || lb.preserve {
		return 0
	}
	r := lb.cur.runs[n-1]
	tr := trimRight(r.text)
	if tr == r.text {
		return 0
	}
	it := lb.items[r.item]
	return lb.c.Measure(it.font, it.size, r.text[len(tr):]).Width
}

// emitLines converts laid-out lines into primitives at offset (ox, oy).
func (c *Ctx) emitLines(l *Layout, items []inlineItem, lines []line, ox, oy, w float32, align mdmodel.Align, code bool) {
	e := &lineEmitter{c: c, l: l, items: items, ox: ox, oy: oy, w: w, align: align, code: code}
	for i := range lines {
		e.line(&lines[i])
		if i > 0 {
			l.Breaks = append(l.Breaks, oy+lines[i].top)
		}
	}
}

// lineEmitter turns runs into primitives.
type lineEmitter struct {
	c      *Ctx
	l      *Layout
	items  []inlineItem
	ox, oy float32
	w      float32
	align  mdmodel.Align
	code   bool
}

func (e *lineEmitter) line(ln *line) {
	shift := float32(0)
	switch e.align {
	case mdmodel.AlignCenter:
		shift = (e.w - ln.width) / 2
	case mdmodel.AlignRight:
		shift = e.w - ln.width
	}
	for _, r := range ln.runs {
		e.run(ln, r, e.ox+shift+r.x)
	}
}

func (e *lineEmitter) add(p Prim) { e.l.Prims = append(e.l.Prims, p) }

func (e *lineEmitter) run(ln *line, r run, x float32) {
	it := e.items[r.item]
	base := e.oy + ln.baseline
	if it.math != nil {
		e.l.Prims = e.c.mathPrims(e.l.Prims, it.math, it.size*mathScale, x, base, it.color)
	} else {
		e.text(it, r, x, base)
	}
	if it.span.Link != "" {
		e.add(Prim{Kind: PLink, X: x, Y: e.oy + ln.top, W: r.w, H: ln.height, URL: it.span.Link})
	}
}

func (e *lineEmitter) text(it inlineItem, r run, x, base float32) {
	if it.span.Mono && !e.code {
		e.codeBackground(it, r, x, base)
	}
	e.add(Prim{Kind: PText, X: x, Y: base, W: r.w, Text: r.text, Font: it.font, Size: it.size, Color: it.color})
	if it.span.Underline {
		e.add(Prim{Kind: PRect, X: x, Y: base + 0.12*it.size, W: r.w, H: max(1, it.size/16), Color: it.color})
	}
	if it.span.Strike {
		e.add(Prim{Kind: PRect, X: x, Y: base - 0.3*it.size, W: r.w, H: max(1, it.size/14), Color: it.color})
	}
}

// codeBackground draws the shaded box behind inline code.
func (e *lineEmitter) codeBackground(it inlineItem, r run, x, base float32) {
	pad := 0.2 * it.size
	tw := r.w
	if txt := trimRight(r.text); txt != r.text {
		tw = e.c.Measure(it.font, it.size, txt).Width
	}
	e.add(Prim{Kind: PRect, X: x - pad, Y: base - r.ascent - pad/2, W: tw + 2*pad, H: r.ascent + r.desc + pad,
		Radius: 0.3 * it.size, Color: e.c.Theme.InlineCodeBg})
}
