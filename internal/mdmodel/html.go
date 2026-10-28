package mdmodel

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
)

var (
	alignOpenRe  = regexp.MustCompile(`(?i)^<(div|p)\s+align\s*=\s*["']?(left|center|right)["']?\s*>$`)
	alignCloseRe = regexp.MustCompile(`(?i)^</(div|p)>$`)
	alignWrapRe  = regexp.MustCompile(`(?is)^<(div|p)\s+align\s*=\s*["']?(left|center|right)["']?\s*>(.*)</(div|p)>$`)
)

func parseAlign(s string) Align {
	switch strings.ToLower(s) {
	case "center":
		return AlignCenter
	case "right":
		return AlignRight
	}
	return AlignLeft
}

// htmlText returns the full text of an HTML block.
func (b *builder) htmlText(n *ast.HTMLBlock) string {
	raw := b.lines(n.Lines())
	if n.HasClosure() {
		raw += "\n" + strings.TrimRight(string(n.ClosureLine.Value(b.src)), "\n")
	}
	return raw
}

// htmlBlock handles raw HTML blocks. Alignment wrappers (<div align="...">,
// as GitHub and Typora use them) align the blocks they contain, comments are
// hidden, and anything else is shown verbatim.
func (b *builder) htmlBlock(n ast.Node, raw string) {
	t := strings.TrimSpace(raw)
	switch {
	case strings.HasPrefix(t, "<!--"):
	case alignOpenRe.MatchString(t):
		b.align = append(b.align, parseAlign(alignOpenRe.FindStringSubmatch(t)[2]))
	case alignCloseRe.MatchString(t) && len(b.align) > 0:
		b.align = b.align[:len(b.align)-1]
	case alignWrapRe.MatchString(t):
		b.alignedHTML(n, alignWrapRe.FindStringSubmatch(t))
	default:
		b.emit(n, Block{Kind: HTML, Spans: []Span{{Text: raw, Mono: true}}}, false)
	}
}

// alignedHTML shows the content of <p align="center">text</p> written on
// consecutive lines.
func (b *builder) alignedHTML(n ast.Node, m []string) {
	b.align = append(b.align, parseAlign(m[2]))
	for _, blk := range Parse([]byte(m[3])).Blocks {
		blk.Align = AlignNone
		blk.Decor = nil
		b.emit(n, blk, false)
	}
	b.align = b.align[:len(b.align)-1]
}
