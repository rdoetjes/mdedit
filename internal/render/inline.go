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
	out := make([]inlineItem, 0, len(spans))
	for _, s := range spans {
		it := inlineItem{
			span:  s,
			font:  FontSpec{Mono: s.Mono, Bold: s.Bold, Italic: s.Italic},
			size:  size,
			color: col,
		}
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
		out = append(out, it)
	}
	return out
}

// tokens splits s into words with their trailing spaces, and lone "\n"
// tokens for hard line breaks.
func tokens(s string) []string {
	var out []string
	start := 0
	inSpace := false
	for i, r := range s {
		switch {
		case r == '\n':
			if i > start {
				out = append(out, s[start:i])
			}
			out = append(out, "\n")
			start = i + 1
			inSpace = false
		case unicode.IsSpace(r):
			inSpace = true
		default:
			if inSpace {
				out = append(out, s[start:i])
				start = i
				inSpace = false
			}
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// layoutInline breaks items into lines no wider than maxW. strut is the
// minimum font size used for line metrics, so lines containing only smaller
// text keep a regular height. With preserve set, leading spaces on lines are
// kept (code blocks); otherwise they are dropped at wrap points.
func (c *Ctx) layoutInline(items []inlineItem, maxW, strut, lineHeight float32, preserve bool) ([]line, float32) {
	strutM := c.Measure(FontSpec{Mono: preserve}, strut, "x")
	if strutM.Ascent == 0 {
		strutM.Ascent, strutM.Descent = 0.9*strut, 0.22*strut
	}
	var lines []line
	var cur line
	x := float32(0)
	y := float32(0)

	finish := func() {
		asc, desc := strutM.Ascent, strutM.Descent
		h := strut * lineHeight
		for _, r := range cur.runs {
			asc = max(asc, r.ascent)
			desc = max(desc, r.desc)
			h = max(h, items[r.item].size*lineHeight)
		}
		h = max(h, asc+desc)
		cur.top = y
		cur.height = h
		cur.baseline = y + (h-(asc+desc))/2 + asc
		cur.width = x
		// Trailing spaces don't count for alignment.
		if n := len(cur.runs); n > 0 && !preserve {
			r := cur.runs[n-1]
			if tr := strings.TrimRightFunc(r.text, unicode.IsSpace); tr != r.text {
				it := items[r.item]
				cur.width -= c.Measure(it.font, it.size, r.text[len(tr):]).Width
			}
		}
		lines = append(lines, cur)
		y += h
		cur = line{}
		x = 0
	}

	place := func(i int, t string, m Metrics) {
		if n := len(cur.runs); n > 0 && cur.runs[n-1].item == i {
			r := &cur.runs[n-1]
			r.text += t
			r.w += m.Width
		} else {
			cur.runs = append(cur.runs, run{item: i, text: t, x: x, w: m.Width, ascent: m.Ascent, desc: m.Descent})
		}
		x += m.Width
	}

	for i, it := range items {
		if it.math != nil {
			// A formula is a single unbreakable box.
			ms := it.size * mathScale
			m := Metrics{Width: it.math.W * ms, Ascent: it.math.H * ms, Descent: it.math.D * ms}
			if x > 0 && x+m.Width > maxW {
				finish()
			}
			place(i, "", m)
			continue
		}
		measure := func(s string) Metrics { return c.Measure(it.font, it.size, s) }
		for _, t := range tokens(it.span.Text) {
			if t == "\n" {
				finish()
				continue
			}
			word := strings.TrimRightFunc(t, unicode.IsSpace)
			wm := measure(word)
			if x > 0 && x+wm.Width > maxW {
				finish()
			}
			if x == 0 && !preserve {
				t = strings.TrimLeftFunc(t, unicode.IsSpace)
				if t == "" {
					continue
				}
				word = strings.TrimRightFunc(t, unicode.IsSpace)
				wm = measure(word)
			}
			// Break words that don't fit on a line of their own.
			for wm.Width > maxW && utf8.RuneCountInString(word) > 1 {
				k := fitRunes(word, maxW-x, measure)
				head := word[:k]
				place(i, head, measure(head))
				finish()
				t = t[k:]
				word = word[k:]
				wm = measure(word)
			}
			place(i, t, measure(t))
		}
	}
	if len(cur.runs) > 0 || len(lines) == 0 {
		finish()
	}
	return lines, y
}

// fitRunes returns the byte length of the longest prefix of s (at least one
// rune) that fits in w.
func fitRunes(s string, w float32, measure func(string) Metrics) int {
	idx := []int{}
	for i := range s {
		if i > 0 {
			idx = append(idx, i)
		}
	}
	idx = append(idx, len(s))
	lo, hi := 0, len(idx)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if measure(s[:idx[mid]]).Width <= w {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return idx[lo]
}

// emitLines converts laid-out lines into primitives at offset (ox, oy).
func (c *Ctx) emitLines(l *Layout, items []inlineItem, lines []line, ox, oy, w float32, align mdmodel.Align, code bool) {
	for li, ln := range lines {
		shift := float32(0)
		switch align {
		case mdmodel.AlignCenter:
			shift = (w - ln.width) / 2
		case mdmodel.AlignRight:
			shift = w - ln.width
		}
		for _, r := range ln.runs {
			it := items[r.item]
			x := ox + shift + r.x
			base := oy + ln.baseline
			if it.math != nil {
				l.Prims = c.mathPrims(l.Prims, it.math, it.size*mathScale, x, base, it.color)
				if it.span.Link != "" {
					l.Prims = append(l.Prims, Prim{Kind: PLink, X: x, Y: oy + ln.top, W: r.w, H: ln.height, URL: it.span.Link})
				}
				continue
			}
			if it.span.Mono && !code {
				pad := 0.2 * it.size
				txt := strings.TrimRightFunc(r.text, unicode.IsSpace)
				tw := r.w
				if txt != r.text {
					tw = c.Measure(it.font, it.size, txt).Width
				}
				l.Prims = append(l.Prims, Prim{Kind: PRect, X: x - pad, Y: base - r.ascent - pad/2,
					W: tw + 2*pad, H: r.ascent + r.desc + pad, Radius: 0.3 * it.size, Color: c.Theme.InlineCodeBg})
			}
			l.Prims = append(l.Prims, Prim{Kind: PText, X: x, Y: base, W: r.w, Text: r.text,
				Font: it.font, Size: it.size, Color: it.color})
			if it.span.Underline {
				l.Prims = append(l.Prims, Prim{Kind: PRect, X: x, Y: base + 0.12*it.size, W: r.w,
					H: max(1, it.size/16), Color: it.color})
			}
			if it.span.Strike {
				l.Prims = append(l.Prims, Prim{Kind: PRect, X: x, Y: base - 0.3*it.size, W: r.w,
					H: max(1, it.size/14), Color: it.color})
			}
			if it.span.Link != "" {
				l.Prims = append(l.Prims, Prim{Kind: PLink, X: x, Y: oy + ln.top, W: r.w, H: ln.height, URL: it.span.Link})
			}
		}
		if li > 0 {
			l.Breaks = append(l.Breaks, oy+ln.top)
		}
	}
}
