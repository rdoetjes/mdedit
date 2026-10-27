// Package mdmodel converts Markdown source into a flat list of styled blocks
// that both the on-screen preview and the PDF exporter render from.
//
// Container blocks (block quotes and list items) are not represented as
// nodes; instead every leaf block carries the stack of containers it lives
// in (Block.Decor). A flat list keeps the preview cheap to virtualise and
// makes page breaking in the PDF exporter straightforward.
package mdmodel

import "image/color"

// Span is a run of inline text with uniform styling.
type Span struct {
	Text      string
	Bold      bool
	Italic    bool
	Mono      bool // inline code, or any text inside a code block
	Strike    bool
	Underline bool
	Link      string      // link target; empty if not a link
	Color     color.NRGBA // syntax-highlight colour; zero means the default colour
}

func (s Span) sameStyle(o Span) bool {
	return s.Bold == o.Bold && s.Italic == o.Italic && s.Mono == o.Mono &&
		s.Strike == o.Strike && s.Underline == o.Underline && s.Link == o.Link && s.Color == o.Color
}

type Kind int

const (
	Paragraph Kind = iota
	Heading
	Code
	Rule
	Table
	Image
	HTML // raw HTML block, shown verbatim
)

type DecorKind int

const (
	Quote DecorKind = iota
	Item
)

// Task states for list items.
const (
	NoTask = iota
	TaskOpen
	TaskDone
)

// Decor is one container (block quote or list item) enclosing a block,
// outermost first.
type Decor struct {
	Kind DecorKind
	// ID identifies the container instance, so renderers can tell whether two
	// consecutive blocks are inside the same quote or item.
	ID int
	// List identifies the list an item belongs to.
	List int
	// ShowMarker is set on the first block of a list item only.
	ShowMarker bool
	Ordered    bool
	Number     string // "3." for ordered items
	Bullet     int    // nesting depth among bulleted lists, 0-based
	Task       int    // NoTask, TaskOpen or TaskDone
}

type Align int

const (
	AlignNone Align = iota
	AlignLeft
	AlignCenter
	AlignRight
)

type TableData struct {
	Align  []Align
	Header [][]Span
	Rows   [][][]Span
}

type Block struct {
	Kind  Kind
	Level int // heading level 1-6
	Spans []Span
	Lang  string // code block language
	Decor []Decor
	Table *TableData
	// Image blocks.
	Src, Alt string
	// ID is the heading anchor ("#overview").
	ID string
	// Line is the 0-based source line the block starts on.
	Line int
	// Align is set for blocks inside <div align="..."> wrappers.
	Align Align
	// Gap is the vertical space above the block, in em.
	Gap float32
}

// InQuote reports whether the block is nested inside a block quote.
func (b *Block) InQuote() bool {
	for _, d := range b.Decor {
		if d.Kind == Quote {
			return true
		}
	}
	return false
}

// Document is the parsed result.
type Document struct {
	Blocks []Block
}

// HeadingIndex returns the index of the block whose anchor is id, or -1.
func (d *Document) HeadingIndex(id string) int {
	for i := range d.Blocks {
		if d.Blocks[i].Kind == Heading && d.Blocks[i].ID == id {
			return i
		}
	}
	return -1
}

// BlockAtLine returns the index of the last block starting at or before the
// given 0-based source line.
func (d *Document) BlockAtLine(line int) int {
	idx := 0
	for i := range d.Blocks {
		if d.Blocks[i].Line > line {
			break
		}
		idx = i
	}
	return idx
}
