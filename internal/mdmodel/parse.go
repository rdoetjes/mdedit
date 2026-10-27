package mdmodel

import (
	"bytes"
	"sort"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
)

// Parse converts Markdown source to a Document.
func Parse(src []byte) *Document {
	root := md.Parser().Parse(text.NewReader(src))
	b := &builder{src: src}
	for i, c := range src {
		if c == '\n' {
			b.lineStarts = append(b.lineStarts, i+1)
		}
	}
	b.block(root)
	b.computeGaps()
	return &Document{Blocks: b.blocks}
}

type builder struct {
	src        []byte
	lineStarts []int // offsets of the starts of lines 1..n
	blocks     []Block
	decor      []Decor
	nextID     int
	lastLine   int
	isTight    []bool // per emitted block: emitted as a tight list paragraph
}

func (b *builder) lineOf(offset int) int {
	return sort.SearchInts(b.lineStarts, offset+1)
}

// firstOffset finds the first source offset covered by n, if any.
func firstOffset(n ast.Node) (int, bool) {
	if n.Type() == ast.TypeBlock {
		if l := n.Lines(); l != nil && l.Len() > 0 {
			return l.At(0).Start, true
		}
	}
	if t, ok := n.(*ast.Text); ok {
		return t.Segment.Start, true
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if off, ok := firstOffset(c); ok {
			return off, true
		}
	}
	return 0, false
}

func (b *builder) emit(n ast.Node, blk Block, tight bool) {
	if off, ok := firstOffset(n); ok {
		b.lastLine = b.lineOf(off)
	}
	blk.Line = b.lastLine
	blk.Decor = append([]Decor(nil), b.decor...)
	b.blocks = append(b.blocks, blk)
	b.isTight = append(b.isTight, tight)
	// Only the first block of a list item shows its marker.
	for i := range b.decor {
		b.decor[i].ShowMarker = false
	}
}

func (b *builder) push(d Decor) {
	b.nextID++
	d.ID = b.nextID
	b.decor = append(b.decor, d)
}

func (b *builder) pop() { b.decor = b.decor[:len(b.decor)-1] }

func (b *builder) block(n ast.Node) {
	switch n := n.(type) {
	case *ast.Blockquote:
		b.push(Decor{Kind: Quote})
		b.children(n)
		b.pop()
	case *ast.List:
		bullet := 0
		for _, d := range b.decor {
			if d.Kind == Item && !d.Ordered {
				bullet++
			}
		}
		num := n.Start
		b.nextID++
		list := b.nextID
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			d := Decor{Kind: Item, List: list, ShowMarker: true, Ordered: n.IsOrdered(), Bullet: bullet}
			if d.Ordered {
				d.Number = strconv.Itoa(num) + string(n.Marker)
				num++
			}
			b.push(d)
			if c.ChildCount() == 0 {
				b.emit(c, Block{Kind: Paragraph}, n.IsTight)
			} else {
				b.children(c)
			}
			b.pop()
		}
	case *ast.Heading:
		blk := Block{Kind: Heading, Level: n.Level, Spans: b.inlines(n)}
		if id, ok := n.AttributeString("id"); ok {
			if v, ok := id.([]byte); ok {
				blk.ID = string(v)
			}
		}
		b.emit(n, blk, false)
	case *ast.Paragraph, *ast.TextBlock:
		_, isText := n.(*ast.TextBlock)
		if img, ok := n.FirstChild().(*ast.Image); ok && n.ChildCount() == 1 {
			b.emit(n, Block{Kind: Image, Src: string(img.Destination), Alt: plainText(img, b.src)}, isText)
			return
		}
		spans := b.inlines(n)
		b.emit(n, Block{Kind: Paragraph, Spans: spans}, isText)
	case *ast.FencedCodeBlock:
		lang := string(n.Language(b.src))
		code := b.lines(n.Lines())
		b.emit(n, Block{Kind: Code, Lang: lang, Spans: Highlight(lang, code)}, false)
	case *ast.CodeBlock:
		code := b.lines(n.Lines())
		b.emit(n, Block{Kind: Code, Spans: Highlight("", code)}, false)
	case *ast.ThematicBreak:
		b.emit(n, Block{Kind: Rule}, false)
	case *ast.HTMLBlock:
		raw := b.lines(n.Lines())
		if n.HasClosure() {
			raw += "\n" + strings.TrimRight(string(n.ClosureLine.Value(b.src)), "\n")
		}
		b.emit(n, Block{Kind: HTML, Spans: []Span{{Text: raw, Mono: true}}}, false)
	case *east.Table:
		b.table(n)
	default:
		b.children(n)
	}
}

func (b *builder) children(n ast.Node) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		b.block(c)
	}
}

func (b *builder) lines(l *text.Segments) string {
	var buf bytes.Buffer
	for i := 0; i < l.Len(); i++ {
		seg := l.At(i)
		buf.Write(seg.Value(b.src))
	}
	return strings.TrimRight(buf.String(), "\n")
}

func (b *builder) table(t *east.Table) {
	td := &TableData{}
	for _, a := range t.Alignments {
		switch a {
		case east.AlignLeft:
			td.Align = append(td.Align, AlignLeft)
		case east.AlignCenter:
			td.Align = append(td.Align, AlignCenter)
		case east.AlignRight:
			td.Align = append(td.Align, AlignRight)
		default:
			td.Align = append(td.Align, AlignNone)
		}
	}
	for row := t.FirstChild(); row != nil; row = row.NextSibling() {
		var cells [][]Span
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			cells = append(cells, b.inlines(cell))
		}
		if _, ok := row.(*east.TableHeader); ok {
			td.Header = cells
		} else {
			td.Rows = append(td.Rows, cells)
		}
	}
	b.emit(t, Block{Kind: Table, Table: td}, false)
}

// computeGaps assigns the vertical space above each block, roughly following
// GitHub's stylesheet (paragraph margins of 1em, headings 1.5em above).
func (b *builder) computeGaps() {
	for i := range b.blocks {
		blk := &b.blocks[i]
		if i == 0 {
			blk.Gap = 0
			continue
		}
		prev := &b.blocks[i-1]
		gap := float32(1)
		switch {
		case blk.Kind == Heading:
			gap = 1.5
		case blk.Kind == Rule || prev.Kind == Rule:
			gap = 1.5
		case b.isTight[i] && sameList(prev, blk):
			// Next item of a tight list, or first item of a nested list.
			gap = 0.25
		}
		blk.Gap = gap
	}
}

// sameList reports whether blk continues the list prev is in: either a
// sibling item, or a list nested inside prev's item.
func sameList(prev, blk *Block) bool {
	for _, d := range blk.Decor {
		if d.Kind != Item {
			continue
		}
		for _, p := range prev.Decor {
			if p.Kind == Item && (p.ID == d.ID || p.List == d.List) {
				return true
			}
		}
	}
	return false
}

// inlines flattens the inline children of n into styled spans.
func (b *builder) inlines(n ast.Node) []Span {
	var out []Span
	b.inline(n, Span{}, &out)
	return mergeSpans(out)
}

func (b *builder) inline(n ast.Node, st Span, out *[]Span) {
	add := func(s Span, t string) {
		if t == "" {
			return
		}
		s.Text = t
		*out = append(*out, s)
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c := c.(type) {
		case *ast.Text:
			v := c.Segment.Value(b.src)
			if !c.IsRaw() {
				v = unescape(v)
			}
			add(st, string(v))
			if c.HardLineBreak() {
				add(st, "\n")
			} else if c.SoftLineBreak() {
				add(st, " ")
			}
		case *ast.String:
			v := c.Value
			if !c.IsRaw() && !c.IsCode() {
				v = unescape(v)
			}
			add(st, string(v))
		case *ast.CodeSpan:
			s := st
			s.Mono = true
			add(s, strings.ReplaceAll(plainText(c, b.src), "\n", " "))
		case *ast.Emphasis:
			s := st
			if c.Level >= 2 {
				s.Bold = true
			} else {
				s.Italic = true
			}
			b.inline(c, s, out)
		case *ast.Link:
			s := st
			s.Link = string(c.Destination)
			b.inline(c, s, out)
		case *ast.AutoLink:
			s := st
			url := string(c.URL(b.src))
			if c.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(url, "mailto:") {
				url = "mailto:" + url
			}
			s.Link = url
			add(s, string(c.Label(b.src)))
		case *ast.Image:
			// Images inside running text are shown by their alt text.
			s := st
			s.Italic = true
			s.Link = string(c.Destination)
			add(s, "[image: "+plainText(c, b.src)+"]")
		case *ast.RawHTML:
			var raw strings.Builder
			for i := 0; i < c.Segments.Len(); i++ {
				seg := c.Segments.At(i)
				raw.Write(seg.Value(b.src))
			}
			tag := strings.ToLower(strings.ReplaceAll(raw.String(), " ", ""))
			if strings.HasPrefix(tag, "<br") {
				add(st, "\n")
			}
			// Other inline tags are dropped; their text content still shows.
		case *east.Strikethrough:
			s := st
			s.Strike = true
			b.inline(c, s, out)
		case *east.TaskCheckBox:
			for i := len(b.decor) - 1; i >= 0; i-- {
				if b.decor[i].Kind == Item {
					if c.IsChecked {
						b.decor[i].Task = TaskDone
					} else {
						b.decor[i].Task = TaskOpen
					}
					break
				}
			}
		default:
			b.inline(c, st, out)
		}
	}
}

func unescape(v []byte) []byte {
	v = util.UnescapePunctuations(v)
	v = util.ResolveNumericReferences(v)
	return util.ResolveEntityNames(v)
}

// plainText concatenates the text of all descendants of n.
func plainText(n ast.Node, src []byte) string {
	var sb strings.Builder
	var walk func(ast.Node)
	walk = func(n ast.Node) {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			switch c := c.(type) {
			case *ast.Text:
				sb.Write(c.Segment.Value(src))
				if c.SoftLineBreak() || c.HardLineBreak() {
					sb.WriteByte('\n')
				}
			case *ast.String:
				sb.Write(c.Value)
			default:
				walk(c)
			}
		}
	}
	walk(n)
	return sb.String()
}

func mergeSpans(in []Span) []Span {
	var out []Span
	for _, s := range in {
		if n := len(out); n > 0 && out[n-1].sameStyle(s) {
			out[n-1].Text += s.Text
			continue
		}
		out = append(out, s)
	}
	return out
}
