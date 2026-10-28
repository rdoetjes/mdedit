package spell

import (
	"math"
	"slices"
	"unicode"

	"mdedit/internal/mdmodel"
)

// Miss is a misspelt word, at rune offsets [Start, End) of the text.
type Miss struct {
	Start, End int
	Word       string
}

// Check returns the misspelt words in a Markdown text, in order. doc is the
// parsed text; its code, maths and HTML blocks are not checked. Neither are
// inline code and maths, HTML tags, link targets, URLs, file names, words
// with digits or underscores, words in capitals (acronyms) and words with
// capitals inside (camelCase, product names).
func Check(d *Dictionary, text []rune, doc *mdmodel.Document) []Miss {
	c := checker{d: d, text: text, skip: skippedLines(doc)}
	c.run()
	return c.misses
}

// lines is a range [from, to) of 0-based source lines.
type lines struct{ from, to int }

// skippedLines returns the source lines of the blocks that aren't prose, in
// order. A block runs until the next one starts.
func skippedLines(doc *mdmodel.Document) []lines {
	var skip []lines
	for i, b := range doc.Blocks {
		if b.Kind != mdmodel.Code && b.Kind != mdmodel.Math && b.Kind != mdmodel.HTML {
			continue
		}
		end := math.MaxInt
		if i+1 < len(doc.Blocks) {
			end = max(b.Line+1, doc.Blocks[i+1].Line)
		}
		skip = append(skip, lines{b.Line, end})
	}
	return skip
}

// checker walks through a text collecting misspellings.
type checker struct {
	d      *Dictionary
	text   []rune
	skip   []lines
	misses []Miss
}

func (c *checker) run() {
	line, start := 0, 0
	for i := 0; i <= len(c.text); i++ {
		if i < len(c.text) && c.text[i] != '\n' {
			continue
		}
		if !c.skipped(line) {
			c.checkLine(start, i)
		}
		line, start = line+1, i+1
	}
}

// skipped reports whether a line is in a skipped block. Lines must be asked
// for in increasing order.
func (c *checker) skipped(line int) bool {
	for len(c.skip) > 0 && c.skip[0].to <= line {
		c.skip = c.skip[1:]
	}
	return len(c.skip) > 0 && c.skip[0].from <= line
}

func (c *checker) checkLine(from, to int) {
	for i := from; i < to; {
		i = c.step(i, to)
	}
}

// step looks at the text at i, on a line ending at to, and returns where to
// continue.
func (c *checker) step(i, to int) int {
	switch r := c.text[i]; {
	case r == '\\':
		return i + 2 // an escaped character
	case r == '`':
		return c.skipCode(i, to)
	case r == '$':
		return c.skipMath(i, to)
	case r == '<':
		return c.skipTag(i, to)
	case r == ']':
		return c.skipLinkTarget(i, to)
	case r == '&':
		return c.skipEntity(i, to)
	case isChunkRune(r):
		return c.chunk(i, to)
	}
	return i + 1
}

// skipCode skips inline code: a run of backticks up to the next run of the
// same length.
func (c *checker) skipCode(i, to int) int {
	n := c.count('`', i, to)
	for j := i + n; j < to; {
		m := c.count('`', j, to)
		switch {
		case m == n:
			return j + m
		case m > 0:
			j += m
		default:
			j++
		}
	}
	return i + n
}

// count returns the number of r in a row at i.
func (c *checker) count(r rune, i, to int) int {
	n := 0
	for i+n < to && c.text[i+n] == r {
		n++
	}
	return n
}

// skipMath skips $inline$ and $$display$$ maths, by the rules of
// mdmodel/math.go: the opening $ is not followed by a space, the closing one
// not preceded by a space nor followed by a digit, so "$5 and $10" is text.
func (c *checker) skipMath(i, to int) int {
	if i+1 < to && c.text[i+1] == '$' {
		if j := c.index("$$", i+2, to); j >= 0 {
			return j + 2
		}
		return i + 2
	}
	if i+1 >= to || unicode.IsSpace(c.text[i+1]) {
		return i + 1
	}
	for j := i + 1; j < to; j++ {
		switch {
		case c.text[j] == '\\':
			j++
		case c.text[j] == '$' && c.closesMath(j, to):
			return j + 1
		}
	}
	return i + 1
}

func (c *checker) closesMath(j, to int) bool {
	return !unicode.IsSpace(c.text[j-1]) && (j+1 >= to || !unicode.IsDigit(c.text[j+1]))
}

// index returns the offset of s in the text between from and to, or -1.
func (c *checker) index(s string, from, to int) int {
	p := []rune(s)
	for j := from; j+len(p) <= to; j++ {
		if slices.Equal(c.text[j:j+len(p)], p) {
			return j
		}
	}
	return -1
}

// skipTag skips an HTML tag, comment or autolink: <…>.
func (c *checker) skipTag(i, to int) int {
	if i+1 >= to {
		return i + 1
	}
	if r := c.text[i+1]; !unicode.IsLetter(r) && r != '/' && r != '!' {
		return i + 1 // "a < b"
	}
	if j := c.index(">", i+1, to); j >= 0 {
		return j + 1
	}
	return i + 1
}

// skipLinkTarget skips the target after a link text: ](url) or ][ref].
func (c *checker) skipLinkTarget(i, to int) int {
	if i+1 >= to {
		return i + 1
	}
	var closer string
	switch c.text[i+1] {
	case '(':
		closer = ")"
	case '[':
		closer = "]"
	default:
		return i + 1
	}
	if j := c.index(closer, i+2, to); j >= 0 {
		return j + 1
	}
	return to
}

// skipEntity skips an HTML entity such as &amp; or &#8212;.
func (c *checker) skipEntity(i, to int) int {
	j := i + 1
	for j < to && j-i <= 10 && (unicode.IsLetter(c.text[j]) || unicode.IsDigit(c.text[j]) || c.text[j] == '#') {
		j++
	}
	if j < to && c.text[j] == ';' {
		return j + 1
	}
	return i + 1
}
