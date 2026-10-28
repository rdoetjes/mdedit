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
)

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM, mathExtension{}),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
)

// Parse converts Markdown source to a Document.
func Parse(src []byte) *Document {
	root := md.Parser().Parse(text.NewReader(src))
	b := &builder{src: src, lineStarts: lineStarts(src)}
	b.block(root)
	b.computeGaps()
	return &Document{Blocks: b.blocks}
}

// lineStarts returns the offsets of the starts of lines 1..n.
func lineStarts(src []byte) []int {
	var out []int
	for i, c := range src {
		if c == '\n' {
			out = append(out, i+1)
		}
	}
	return out
}

type builder struct {
	src        []byte
	lineStarts []int // offsets of the starts of lines 1..n
	blocks     []Block
	decor      []Decor
	nextID     int
	lastLine   int
	isTight    []bool  // per emitted block: emitted as a tight list paragraph
	align      []Align // open <div align="..."> wrappers
}

func (b *builder) lineOf(offset int) int {
	return sort.SearchInts(b.lineStarts, offset+1)
}

// firstOffset finds the first source offset covered by n, if any.
func firstOffset(n ast.Node) (int, bool) {
	if n.Type() == ast.TypeBlock && n.Lines() != nil && n.Lines().Len() > 0 {
		return n.Lines().At(0).Start, true
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

// emit adds a block, recording its source line and enclosing containers.
func (b *builder) emit(n ast.Node, blk Block, tight bool) {
	if off, ok := firstOffset(n); ok {
		b.lastLine = b.lineOf(off)
	}
	blk.Line = b.lastLine
	if n := len(b.align); n > 0 && blk.Align == AlignNone {
		blk.Align = b.align[n-1]
	}
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

// block converts a block node and its descendants.
func (b *builder) block(n ast.Node) {
	switch n := n.(type) {
	case *ast.Blockquote:
		b.quote(n)
	case *ast.List:
		b.list(n)
	case *ast.Heading:
		b.heading(n)
	case *ast.Paragraph, *ast.TextBlock:
		b.paragraph(n)
	case *mathBlock:
		b.emit(n, mathBlockOf(b.lines(n.Lines())), false)
	case *ast.FencedCodeBlock:
		b.fencedCode(n)
	case *ast.CodeBlock:
		b.emit(n, Block{Kind: Code, Spans: Highlight("", b.lines(n.Lines()))}, false)
	case *ast.ThematicBreak:
		b.emit(n, Block{Kind: Rule}, false)
	case *ast.HTMLBlock:
		b.htmlBlock(n, b.htmlText(n))
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

func (b *builder) quote(n *ast.Blockquote) {
	b.push(Decor{Kind: Quote})
	b.children(n)
	b.pop()
}

func (b *builder) list(n *ast.List) {
	b.nextID++
	d := Decor{Kind: Item, List: b.nextID, ShowMarker: true, Ordered: n.IsOrdered(), Bullet: b.bulletDepth()}
	num := n.Start
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if d.Ordered {
			d.Number = strconv.Itoa(num) + string(n.Marker)
			num++
		}
		b.listItem(c, d, n.IsTight)
	}
}

// bulletDepth counts the bulleted lists the current block is nested in.
func (b *builder) bulletDepth() int {
	n := 0
	for _, d := range b.decor {
		if d.Kind == Item && !d.Ordered {
			n++
		}
	}
	return n
}

func (b *builder) listItem(item ast.Node, d Decor, tight bool) {
	b.push(d)
	if item.ChildCount() == 0 {
		b.emit(item, Block{Kind: Paragraph}, tight)
	} else {
		b.children(item)
	}
	b.pop()
}

func (b *builder) heading(n *ast.Heading) {
	blk := Block{Kind: Heading, Level: n.Level, Spans: b.inlines(n)}
	if id, ok := n.AttributeString("id"); ok {
		if v, ok := id.([]byte); ok {
			blk.ID = string(v)
		}
	}
	b.emit(n, blk, false)
}

// paragraph converts a paragraph; one consisting of a single image becomes
// an image block.
func (b *builder) paragraph(n ast.Node) {
	_, tight := n.(*ast.TextBlock)
	if img, ok := n.FirstChild().(*ast.Image); ok && n.ChildCount() == 1 {
		b.emit(n, Block{Kind: Image, Src: string(img.Destination), Alt: plainText(img, b.src)}, tight)
		return
	}
	b.emit(n, Block{Kind: Paragraph, Spans: b.inlines(n)}, tight)
}

func (b *builder) fencedCode(n *ast.FencedCodeBlock) {
	lang := string(n.Language(b.src))
	code := b.lines(n.Lines())
	if lang == "math" { // GitHub's ```math blocks
		b.emit(n, mathBlockOf(code), false)
		return
	}
	b.emit(n, Block{Kind: Code, Lang: lang, Spans: Highlight(lang, code)}, false)
}

func mathBlockOf(tex string) Block {
	return Block{Kind: Math, Spans: []Span{{Text: tex, Math: true, Display: true}}}
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
	td := &TableData{Align: tableAlignments(t.Alignments)}
	for row := t.FirstChild(); row != nil; row = row.NextSibling() {
		cells := b.tableCells(row)
		if _, ok := row.(*east.TableHeader); ok {
			td.Header = cells
		} else {
			td.Rows = append(td.Rows, cells)
		}
	}
	b.emit(t, Block{Kind: Table, Table: td}, false)
}

func (b *builder) tableCells(row ast.Node) [][]Span {
	var cells [][]Span
	for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
		cells = append(cells, b.inlines(cell))
	}
	return cells
}

func tableAlignments(in []east.Alignment) []Align {
	out := make([]Align, len(in))
	for i, a := range in {
		switch a {
		case east.AlignLeft:
			out[i] = AlignLeft
		case east.AlignCenter:
			out[i] = AlignCenter
		case east.AlignRight:
			out[i] = AlignRight
		}
	}
	return out
}

// computeGaps assigns the vertical space above each block, roughly following
// GitHub's stylesheet (paragraph margins of 1em, headings 1.5em above).
func (b *builder) computeGaps() {
	for i := 1; i < len(b.blocks); i++ {
		b.blocks[i].Gap = b.gapBetween(i-1, i)
	}
}

func (b *builder) gapBetween(prevIdx, i int) float32 {
	prev, blk := &b.blocks[prevIdx], &b.blocks[i]
	switch {
	case blk.Kind == Heading, blk.Kind == Rule, prev.Kind == Rule:
		return 1.5
	case b.isTight[i] && sameList(prev, blk):
		// Next item of a tight list, or first item of a nested list.
		return 0.25
	}
	return 1
}

// sameList reports whether blk continues the list prev is in: either a
// sibling item, or a list nested inside prev's item.
func sameList(prev, blk *Block) bool {
	for _, d := range blk.Decor {
		if d.Kind == Item && inItemOf(prev, d) {
			return true
		}
	}
	return false
}

// inItemOf reports whether blk is in item d or another item of its list.
func inItemOf(blk *Block, d Decor) bool {
	for _, p := range blk.Decor {
		if p.Kind == Item && (p.ID == d.ID || p.List == d.List) {
			return true
		}
	}
	return false
}
