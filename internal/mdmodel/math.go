package mdmodel

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Maths syntax as in Typora and Pandoc:
//
//	$$                      $$ display maths on a line of its own $$
//	\frac{a}{b}
//	$$
//
// and $inline maths$, where the opening $ must not be followed by a space
// and the closing $ must not be preceded by a space or followed by a digit,
// so "$5 and $10" stays text.

var (
	kindMathBlock  = ast.NewNodeKind("MathBlock")
	kindMathInline = ast.NewNodeKind("MathInline")
)

type mathBlock struct {
	ast.BaseBlock
	closed bool
}

func (n *mathBlock) Kind() ast.NodeKind { return kindMathBlock }
func (n *mathBlock) IsRaw() bool        { return true }
func (n *mathBlock) Dump(src []byte, level int) {
	ast.DumpHelper(n, src, level, nil, nil)
}

type mathInline struct {
	ast.BaseInline
	value   []byte
	display bool // $$…$$ inside a paragraph
}

func (n *mathInline) Kind() ast.NodeKind { return kindMathInline }
func (n *mathInline) Dump(src []byte, level int) {
	ast.DumpHelper(n, src, level, map[string]string{"Value": string(n.value)}, nil)
}

type mathBlockParser struct{}

func (mathBlockParser) Trigger() []byte { return []byte{'$'} }

func (mathBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, seg := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || !bytes.HasPrefix(line[pos:], []byte("$$")) {
		return nil, parser.NoChildren
	}
	node := &mathBlock{}
	rest := line[pos+2:]
	start := seg.Start + pos + 2
	trimmed := util.TrimRightSpace(rest)
	// $$ formula $$ on one line.
	if len(trimmed) >= 2 && bytes.HasSuffix(trimmed, []byte("$$")) {
		inner := trimmed[:len(trimmed)-2]
		node.Lines().Append(text.NewSegment(start, start+len(inner)))
		node.closed = true
		reader.AdvanceToEOL()
		return node, parser.NoChildren
	}
	if !util.IsBlank(rest) {
		node.Lines().Append(text.NewSegment(start, seg.Stop))
	}
	reader.AdvanceToEOL()
	return node, parser.NoChildren
}

func (mathBlockParser) Continue(n ast.Node, reader text.Reader, pc parser.Context) parser.State {
	node := n.(*mathBlock)
	if node.closed {
		return parser.Close
	}
	line, seg := reader.PeekLine()
	if line == nil {
		return parser.Close
	}
	trimmed := util.TrimRightSpace(line)
	if bytes.HasSuffix(trimmed, []byte("$$")) {
		inner := trimmed[:len(trimmed)-2]
		if !util.IsBlank(inner) {
			node.Lines().Append(text.NewSegment(seg.Start, seg.Start+len(inner)))
		}
		reader.AdvanceToEOL()
		return parser.Close
	}
	node.Lines().Append(seg)
	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

func (mathBlockParser) Close(ast.Node, text.Reader, parser.Context) {}
func (mathBlockParser) CanInterruptParagraph() bool                 { return true }
func (mathBlockParser) CanAcceptIndentedLine() bool                 { return false }

type mathInlineParser struct{}

func (mathInlineParser) Trigger() []byte { return []byte{'$'} }

func (mathInlineParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	if len(line) < 2 {
		return nil
	}
	if line[1] == '$' {
		// $$…$$ inside a paragraph: display style.
		end := bytes.Index(line[2:], []byte("$$"))
		if end <= 0 {
			return nil
		}
		v := line[2 : 2+end]
		block.Advance(end + 4)
		return &mathInline{value: append([]byte(nil), v...), display: true}
	}
	if isSpaceByte(line[1]) {
		return nil
	}
	for i := 1; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++ // skip escaped character, e.g. \$
		case '$':
			if isSpaceByte(line[i-1]) || (i+1 < len(line) && line[i+1] >= '0' && line[i+1] <= '9') {
				continue
			}
			v := line[1:i]
			block.Advance(i + 1)
			return &mathInline{value: append([]byte(nil), v...)}
		}
	}
	return nil
}

func isSpaceByte(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

type mathExtension struct{}

func (mathExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(util.Prioritized(mathBlockParser{}, 750)),
		parser.WithInlineParsers(util.Prioritized(mathInlineParser{}, 501)),
	)
}
