package mdmodel

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/util"
)

// inlines flattens the inline children of n into styled spans.
func (b *builder) inlines(n ast.Node) []Span {
	var out []Span
	b.inline(n, Span{}, &out)
	return mergeSpans(out)
}

// inline converts the inline children of n, styled as st, appending to out.
func (b *builder) inline(n ast.Node, st Span, out *[]Span) {
	r := &inlineRun{b: b, out: out, base: st, st: st}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		r.node(c)
	}
}

// inlineRun converts a sequence of sibling inline nodes.
type inlineRun struct {
	b    *builder
	out  *[]Span
	base Span // style of the container
	st   Span // current style; raw HTML tags such as <u> change it
}

func (r *inlineRun) add(s Span, t string) {
	if t != "" {
		s.Text = t
		*r.out = append(*r.out, s)
	}
}

// nested converts the children of n with style s.
func (r *inlineRun) nested(n ast.Node, s Span) { r.b.inline(n, s, r.out) }

func (r *inlineRun) node(c ast.Node) {
	switch c := c.(type) {
	case *ast.Text:
		r.text(c)
	case *ast.String:
		r.str(c)
	case *ast.CodeSpan:
		r.codeSpan(c)
	case *ast.Emphasis:
		r.emphasis(c)
	case *ast.Link:
		r.link(c)
	case *ast.AutoLink:
		r.autoLink(c)
	case *ast.Image:
		r.image(c)
	case *ast.RawHTML:
		r.rawHTML(c)
	case *mathInline:
		r.math(c)
	case *east.Strikethrough:
		r.strike(c)
	case *east.TaskCheckBox:
		r.b.markTask(c.IsChecked)
	default:
		r.nested(c, r.st)
	}
}

func (r *inlineRun) text(c *ast.Text) {
	v := c.Segment.Value(r.b.src)
	if !c.IsRaw() {
		v = unescape(v)
	}
	r.add(r.st, string(v))
	if c.HardLineBreak() {
		r.add(r.st, "\n")
	} else if c.SoftLineBreak() {
		r.add(r.st, " ")
	}
}

func (r *inlineRun) str(c *ast.String) {
	v := c.Value
	if !c.IsRaw() && !c.IsCode() {
		v = unescape(v)
	}
	r.add(r.st, string(v))
}

func (r *inlineRun) codeSpan(c *ast.CodeSpan) {
	s := r.st
	s.Mono = true
	r.add(s, strings.ReplaceAll(plainText(c, r.b.src), "\n", " "))
}

func (r *inlineRun) emphasis(c *ast.Emphasis) {
	s := r.st
	if c.Level >= 2 {
		s.Bold = true
	} else {
		s.Italic = true
	}
	r.nested(c, s)
}

func (r *inlineRun) link(c *ast.Link) {
	s := r.st
	s.Link = string(c.Destination)
	r.nested(c, s)
}

func (r *inlineRun) autoLink(c *ast.AutoLink) {
	s := r.st
	s.Link = string(c.URL(r.b.src))
	if c.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(s.Link, "mailto:") {
		s.Link = "mailto:" + s.Link
	}
	r.add(s, string(c.Label(r.b.src)))
}

// image shows an image inside running text by its alt text.
func (r *inlineRun) image(c *ast.Image) {
	s := r.st
	s.Italic = true
	s.Link = string(c.Destination)
	r.add(s, "[image: "+plainText(c, r.b.src)+"]")
}

func (r *inlineRun) math(c *mathInline) {
	s := r.st
	s.Math, s.Display = true, c.display
	s.Text = string(c.value)
	*r.out = append(*r.out, s)
}

func (r *inlineRun) strike(c *east.Strikethrough) {
	s := r.st
	s.Strike = true
	r.nested(c, s)
}

// rawHTML handles inline tags: <br> breaks the line, and simple formatting
// tags style the text that follows them until the closing tag. Other tags
// are dropped but their text shows.
func (r *inlineRun) rawHTML(c *ast.RawHTML) {
	tag := r.b.rawText(c)
	if strings.HasPrefix(tag, "<br") {
		r.add(r.st, "\n")
		return
	}
	closing := strings.HasPrefix(tag, "</")
	name, _, _ := strings.Cut(strings.Trim(tag, "</>"), "/")
	r.applyTag(name, closing)
}

// applyTag switches a style on for an opening tag, or back to the
// container's style for a closing one.
func (r *inlineRun) applyTag(name string, closing bool) {
	switch name {
	case "u", "ins":
		r.st.Underline = !closing || r.base.Underline
	case "b", "strong":
		r.st.Bold = !closing || r.base.Bold
	case "i", "em":
		r.st.Italic = !closing || r.base.Italic
	case "s", "del", "strike":
		r.st.Strike = !closing || r.base.Strike
	case "code", "kbd":
		r.st.Mono = !closing || r.base.Mono
	}
}

// rawText returns an inline HTML tag, lower-cased and without spaces.
func (b *builder) rawText(c *ast.RawHTML) string {
	var raw strings.Builder
	for i := 0; i < c.Segments.Len(); i++ {
		seg := c.Segments.At(i)
		raw.Write(seg.Value(b.src))
	}
	return strings.ToLower(strings.ReplaceAll(raw.String(), " ", ""))
}

// markTask records a task-list checkbox on the innermost list item.
func (b *builder) markTask(checked bool) {
	for i := len(b.decor) - 1; i >= 0; i-- {
		if b.decor[i].Kind != Item {
			continue
		}
		b.decor[i].Task = TaskOpen
		if checked {
			b.decor[i].Task = TaskDone
		}
		return
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
	writePlainText(&sb, n, src)
	return sb.String()
}

func writePlainText(sb *strings.Builder, n ast.Node, src []byte) {
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
			writePlainText(sb, c, src)
		}
	}
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
