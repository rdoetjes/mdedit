package edit

import (
	"regexp"
	"strings"
)

// Alignment values for SetAlign.
const (
	AlignLeft   = "left"
	AlignCenter = "center"
	AlignRight  = "right"
)

var (
	alignRe = regexp.MustCompile(`^\s*<(div|p)\s+align\s*=\s*["']?(left|center|right)["']?\s*>\s*$`)
	closeRe = regexp.MustCompile(`^\s*</(div|p)>\s*$`)
)

// SetAlign aligns the paragraph at the selection by wrapping it in
// <div align="...">, the convention GitHub and Typora render. The blank
// lines inside the div keep its content parsed as Markdown. Left alignment
// removes the wrapper.
func SetAlign(s State, align string) State {
	d := newLineDoc(s)
	first, last := d.selectedLines()
	open, close := d.wrapperAround(first, last)
	switch {
	case open >= 0 && close >= 0 && align == AlignLeft:
		return d.unwrap(open, close)
	case open >= 0 && close >= 0:
		return d.retag(open, align)
	case align == AlignLeft:
		return s
	}
	return d.wrap(first, last, align)
}

// lineDoc is a state split into lines, with the rune offset of each line.
type lineDoc struct {
	s       State
	lines   []string
	offsets []int // offsets[i] is the start of line i; one extra at the end
}

func newLineDoc(s State) *lineDoc {
	lines := strings.Split(string(s.Text), "\n")
	offsets := make([]int, len(lines)+1)
	for i, l := range lines {
		offsets[i+1] = offsets[i] + len([]rune(l)) + 1
	}
	return &lineDoc{s: s, lines: lines, offsets: offsets}
}

func (d *lineDoc) lineOf(pos int) int {
	for i := range d.lines {
		if pos < d.offsets[i+1] {
			return i
		}
	}
	return len(d.lines) - 1
}

// selectedLines returns the first and last line touched by the selection.
func (d *lineDoc) selectedLines() (int, int) {
	s := d.s
	first, last := d.lineOf(s.Start), d.lineOf(max(s.Start, s.End-1))
	if s.End > s.Start && s.End == d.offsets[last] {
		last--
	}
	return first, last
}

func (d *lineDoc) blank(i int) bool { return strings.TrimSpace(d.lines[i]) == "" }

// wrapperAround finds the opening and closing lines of an alignment div
// enclosing lines first..last, or -1.
func (d *lineDoc) wrapperAround(first, last int) (open, close int) {
	open = d.openAbove(first)
	if open < 0 {
		return -1, -1
	}
	return open, d.closeBelow(max(last, open+1))
}

func (d *lineDoc) openAbove(first int) int {
	for i := first; i >= 0; i-- {
		if alignRe.MatchString(d.lines[i]) {
			return i
		}
		if closeRe.MatchString(d.lines[i]) && i != first {
			return -1
		}
	}
	return -1
}

func (d *lineDoc) closeBelow(from int) int {
	for i := from; i < len(d.lines); i++ {
		if closeRe.MatchString(d.lines[i]) {
			return i
		}
		if alignRe.MatchString(d.lines[i]) {
			return -1
		}
	}
	return -1
}

// unwrap removes the closing line (and a blank line before it), then the
// opening line (and a blank line after it).
func (d *lineDoc) unwrap(open, close int) State {
	cFrom := close
	if cFrom-1 > open && d.blank(cFrom-1) {
		cFrom--
	}
	oTo := open
	if oTo+1 < cFrom && d.blank(oTo+1) {
		oTo++
	}
	o := d.offsets
	t := d.s.replace(o[cFrom]-1, min(o[close+1]-1, len(d.s.Text)), "", 0, 0)
	removed := o[oTo+1] - o[open]
	start, end := d.s.Start, d.s.End
	if start >= o[oTo+1] {
		start -= removed
		end -= removed
	}
	return t.replace(o[open], o[oTo+1], "", max(start, o[open]), max(end, o[open]))
}

// retag changes the alignment of an existing wrapper.
func (d *lineDoc) retag(open int, align string) State {
	tag := `<div align="` + align + `">`
	old := len([]rune(d.lines[open]))
	by := len([]rune(tag)) - old
	start := d.offsets[open]
	return d.s.replace(start, start+old, tag, d.shift(d.s.Start, open+1, by), d.shift(d.s.End, open+1, by))
}

// shift moves pos by `by` if it is at or after the start of line.
func (d *lineDoc) shift(pos, line, by int) int {
	if pos >= d.offsets[line] {
		return pos + by
	}
	return pos
}

// wrap puts the paragraph around lines first..last in a new wrapper.
func (d *lineDoc) wrap(first, last int, align string) State {
	for first > 0 && !d.blank(first-1) {
		first--
	}
	for last < len(d.lines)-1 && !d.blank(last+1) {
		last++
	}
	before := `<div align="` + align + `">` + "\n\n"
	endOff := d.offsets[last] + len([]rune(d.lines[last]))
	t := d.s.replace(endOff, endOff, "\n\n</div>", 0, 0)
	by := len([]rune(before))
	return t.replace(d.offsets[first], d.offsets[first], before, d.s.Start+by, d.s.End+by)
}
