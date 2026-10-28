package edit

import (
	"strings"
	"testing"

	"mdedit/internal/mdmodel"
)

// st builds a State from text where "|" marks the caret, or "[" and "]"
// mark a selection.
func st(t *testing.T, s string) State {
	t.Helper()
	if i := strings.Index(s, "|"); i >= 0 {
		pos := len([]rune(s[:i]))
		return New(s[:i]+s[i+1:], pos, pos)
	}
	a, b := strings.Index(s, "["), strings.Index(s, "]")
	if a < 0 || b < a {
		t.Fatalf("bad fixture %q", s)
	}
	text := s[:a] + s[a+1:b] + s[b+1:]
	return New(text, len([]rune(s[:a])), len([]rune(s[:b]))-1)
}

// show renders a State back into fixture notation.
func show(s State) string {
	r := s.Text
	if s.Start == s.End {
		return string(r[:s.Start]) + "|" + string(r[s.Start:])
	}
	return string(r[:s.Start]) + "[" + string(r[s.Start:s.End]) + "]" + string(r[s.End:])
}

func check(t *testing.T, name string, got State, want string) {
	t.Helper()
	if show(got) != want {
		t.Errorf("%s:\n got %q\nwant %q", name, show(got), want)
	}
}

func TestInline(t *testing.T) {
	check(t, "bold selection", ToggleInline(st(t, "a [word] b"), Bold), "a **[word]** b")
	check(t, "unbold outside", ToggleInline(st(t, "a **[word]** b"), Bold), "a [word] b")
	check(t, "unbold inside", ToggleInline(st(t, "a [**word**] b"), Bold), "a [word] b")
	check(t, "bold word at caret", ToggleInline(st(t, "a wo|rd b"), Bold), "a **[word]** b")
	check(t, "bold empty", ToggleInline(st(t, "a | b"), Bold), "a **|** b")
	check(t, "italic inside bold", ToggleInline(st(t, "**[x]**"), Italic), "***[x]***")
	check(t, "unitalic of bold-italic", ToggleInline(st(t, "***[x]***"), Italic), "**[x]**")
	check(t, "italic doesn't unwrap bold", ToggleInline(st(t, "[**x**]"), Italic), "*[**x**]*")
	check(t, "trim spaces", ToggleInline(st(t, "a[ word ]b"), Italic), "a *[word]* b")
	check(t, "underline", ToggleInline(st(t, "[x]"), Underline), "<u>[x]</u>")
	check(t, "un-underline", ToggleInline(st(t, "<u>[x]</u>"), Underline), "[x]")
}

func TestLists(t *testing.T) {
	check(t, "bullet line", ToggleList(st(t, "fo|o"), Bullets), "- fo|o")
	check(t, "unbullet line", ToggleList(st(t, "- fo|o"), Bullets), "fo|o")
	check(t, "number lines", ToggleList(st(t, "[a\nb\nc]"), Numbers), "[1. a\n2. b\n3. c]")
	check(t, "bullets to numbers", ToggleList(st(t, "[- a\n- b]"), Numbers), "[1. a\n2. b]")
	check(t, "task", ToggleList(st(t, "x|"), Tasks), "- [ ] x|")
	check(t, "quote", ToggleQuote(st(t, "[a\n\nb]")), "[> a\n>\n> b]")
	check(t, "unquote", ToggleQuote(st(t, "> a|")), "a|")
}

func TestHeading(t *testing.T) {
	check(t, "h2", SetHeading(st(t, "ti|tle"), 2), "## ti|tle")
	check(t, "h2 to h3", SetHeading(st(t, "## ti|tle"), 3), "### ti|tle")
	check(t, "toggle off", SetHeading(st(t, "## ti|tle"), 2), "ti|tle")
	check(t, "paragraph", SetHeading(st(t, "# ti|tle"), 0), "ti|tle")
}

func TestNewline(t *testing.T) {
	check(t, "plain", Newline(st(t, "abc|")), "abc\n|")
	check(t, "bullet", Newline(st(t, "- item|")), "- item\n- |")
	check(t, "star bullet", Newline(st(t, "*   item|")), "*   item\n*   |")
	check(t, "numbered", Newline(st(t, "1. one|")), "1. one\n2. |")
	check(t, "paren numbered", Newline(st(t, "9) nine|")), "9) nine\n10) |")
	check(t, "nested", Newline(st(t, "- a\n  - b|")), "- a\n  - b\n  - |")
	check(t, "task", Newline(st(t, "- [x] done|")), "- [x] done\n- [ ] |")
	check(t, "split item", Newline(st(t, "- ab|cd")), "- ab\n- |cd")
	check(t, "end list", Newline(st(t, "1. one\n2. |")), "1. one\n\n|")
	check(t, "end list mid doc", Newline(st(t, "- a\n- |\nnext")), "- a\n\n|\nnext")
	check(t, "quote", Newline(st(t, "> hi|")), "> hi\n> |")
	check(t, "end quote", Newline(st(t, "> hi\n> |")), "> hi\n\n|")
	check(t, "caret in marker", Newline(st(t, "|- a")), "\n|- a")
	check(t, "selection replaced", Newline(st(t, "- a[bc]")), "- a\n- |")
}

func TestAlign(t *testing.T) {
	check(t, "center", SetAlign(st(t, "x\n\npa|ra\ngraph\n\ny"), AlignCenter),
		"x\n\n<div align=\"center\">\n\npa|ra\ngraph\n\n</div>\n\ny")
	check(t, "center to right", SetAlign(st(t, "<div align=\"center\">\n\npa|ra\n\n</div>"), AlignRight),
		"<div align=\"right\">\n\npa|ra\n\n</div>")
	check(t, "back to left", SetAlign(st(t, "x\n\n<div align=\"center\">\n\npa|ra\n\n</div>\n\ny"), AlignLeft),
		"x\n\npa|ra\n\ny")
	check(t, "left noop", SetAlign(st(t, "pa|ra"), AlignLeft), "pa|ra")
}

func TestInsert(t *testing.T) {
	check(t, "table on empty doc", Table(st(t, "|"), 2, 1),
		"| |    |     |\n| --- | --- |\n|     |     |\n")
	check(t, "table after text", Table(st(t, "text|"), 1, 0),
		"text\n\n| |    |\n| --- |\n")
	check(t, "table before text", Table(st(t, "a|\nb"), 1, 0),
		"a\n\n| |    |\n| --- |\n\nb")
	check(t, "code block", CodeBlock(st(t, "|")), "```|\n\n```\n")
	check(t, "fence selection", CodeBlock(st(t, "[x\ny]")), "```|\nx\ny\n```")
	check(t, "link", Link(st(t, "[go]"), false), "[go]([https://])")
	check(t, "link from url", Link(st(t, "[https://go.dev]"), false), "[|](https://go.dev)")
	check(t, "image", Link(st(t, "|"), true), "![[alt text]](image.png)")
}

func TestTabs(t *testing.T) {
	check(t, "expand", ExpandTabs(st(t, "a\tb\n\tc|")), "a   b\n    c|")
	check(t, "expand keeps selection", ExpandTabs(st(t, "\t[x]\ty")), "    [x]   y")
	check(t, "tab inserts", Tab(st(t, "ab|")), "ab  |")
	check(t, "tab at stop", Tab(st(t, "abcd|")), "abcd    |")
	check(t, "tab nests item", Tab(st(t, "- a\n- b|")), "- a\n    - b|")
	check(t, "tab indents lines", Tab(st(t, "[x\ny]")), "[    x\n    y]")
	check(t, "outdent", Outdent(st(t, "- a\n    - b|")), "- a\n- b|")
	check(t, "outdent partial", Outdent(st(t, "  x|")), "x|")
}

func TestEmptyLines(t *testing.T) {
	check(t, "h1 on empty line", SetHeading(st(t, "|"), 1), "# |")
	check(t, "h3 on empty line", SetHeading(st(t, "a\n|\nb"), 3), "a\n### |\nb")
	check(t, "h2 to h4 before typing", SetHeading(st(t, "## |"), 4), "#### |")
	check(t, "h2 toggles off before typing", SetHeading(st(t, "## |"), 2), "|")
	check(t, "blank lines in selection untouched", SetHeading(st(t, "[a\n\nb]"), 2), "[## a\n\n## b]")
	check(t, "bullet on empty line", ToggleList(st(t, "|"), Bullets), "- |")
	check(t, "number on empty line", ToggleList(st(t, "|"), Numbers), "1. |")
	check(t, "task on empty line", ToggleList(st(t, "|"), Tasks), "- [ ] |")
	check(t, "quote on empty line", ToggleQuote(st(t, "|")), "> |")
}

func TestTOC(t *testing.T) {
	doc := "# Title\n\n## Intro *text*\n\n### Detail\n\n## Intro text\n\n#### Deep\n\n> ## Quoted\n"
	want := "- [Title](#title)\n" +
		"  - [Intro text](#intro-text)\n" +
		"    - [Detail](#detail)\n" +
		"  - [Intro text](#intro-text-1)\n" +
		"    - [Deep](#deep)"
	if got := BuildTOC(doc); got != want {
		t.Errorf("BuildTOC:\n%s\nwant:\n%s", got, want)
	}
	// Skipped levels at the start don't create code blocks.
	if got := BuildTOC("### a\n\n# b\n"); got != "- [a](#a)\n- [b](#b)" {
		t.Errorf("skipped levels: %q", got)
	}

	s := InsertTOC(st(t, "|\n\n# A\n\n## B\n"))
	check(t, "insert", s, "<!-- toc -->\n\n- [A](#a)\n  - [B](#b)\n\n<!-- /toc -->|\n\n# A\n\n## B\n")

	// Update after adding a heading; the caret after the TOC moves with it.
	upd := UpdateTOC(st(t, "<!-- toc -->\n\n- [A](#a)\n\n<!-- /toc -->\n\n# A\n\n# C|\n"))
	check(t, "update", upd, "<!-- toc -->\n\n- [A](#a)\n- [C](#c)\n\n<!-- /toc -->\n\n# A\n\n# C|\n")

	// Inserting when one exists updates it instead of adding a second.
	again := InsertTOC(st(t, "<!-- toc -->\n<!-- /toc -->\n\n# X|"))
	if strings.Count(again.String(), TOCStart) != 1 || !strings.Contains(again.String(), "[X](#x)") {
		t.Errorf("second insert: %q", again.String())
	}

	// The TOC renders as a list of links with the markers hidden.
	d := mdmodel.Parse([]byte(s.String()))
	if len(d.Blocks) == 0 || d.Blocks[0].Kind != mdmodel.Paragraph || d.Blocks[0].Spans[0].Link != "#a" {
		t.Errorf("rendered TOC: %+v", d.Blocks[0])
	}
	if d.HeadingIndex("b") < 0 {
		t.Error("anchor #b missing")
	}
}

func TestMath(t *testing.T) {
	check(t, "inline", ToggleInline(st(t, "a [x^2] b"), InlineMath), "a $[x^2]$ b")
	check(t, "un-inline", ToggleInline(st(t, "a $[x^2]$ b"), InlineMath), "a [x^2] b")
	check(t, "block", MathBlock(st(t, "text|")), "text\n\n$$\n|\n$$\n")
	check(t, "wrap block", MathBlock(st(t, "[a = b]")), "$$\n[a = b]\n$$")
}

// Regression: markers mentioned in prose or code must not be taken for a
// table of contents (README.md describes them inline).
func TestTOCMarkersInText(t *testing.T) {
	for _, doc := range []string{
		"# A\n\nUse `<!-- toc -->` and `<!-- /toc -->` markers.\n\n## B\n",
		"# A\n\n```\n<!-- toc -->\n<!-- /toc -->\n```\n\n## B\n",
		"# A\n\n~~~md\n<!-- toc -->\n~~~\n<!-- /toc -->\n\n## B\n",
	} {
		s := New(doc, 3, 3) // caret at the end of "# A"
		if _, _, ok := FindTOC(s); ok {
			t.Errorf("FindTOC found a TOC in %q", doc)
		}
		out := InsertTOC(s).String()
		if !strings.HasPrefix(out, "# A\n\n"+TOCStart+"\n\n- [A](#a)\n  - [B](#b)\n\n"+TOCEnd+"\n\n") {
			t.Errorf("TOC not inserted after the title:\n%s", out)
		}
		if !strings.HasSuffix(out, doc[3:]) {
			t.Errorf("the rest of the document changed:\n%s", out)
		}
	}
	// A real TOC is still found even when the text also mentions markers.
	doc := "Use `<!-- toc -->` here.\n\n  <!-- toc -->\n- old\n<!-- /toc -->\n\n# X\n"
	upd := UpdateTOC(New(doc, 0, 0)).String()
	if !strings.Contains(upd, "Use `<!-- toc -->` here.") || !strings.Contains(upd, "- [X](#x)") || strings.Contains(upd, "- old") {
		t.Errorf("update with mentions:\n%s", upd)
	}
}

func TestVideo(t *testing.T) {
	got := Video(st(t, "Intro|"), "dQw4w9WgXcQ", "Talk").String()
	want := "Intro\n\n[![Talk](https://img.youtube.com/vi/dQw4w9WgXcQ/maxresdefault.jpg)](https://www.youtube.com/watch?v=dQw4w9WgXcQ)\n"
	if got != want {
		t.Errorf("got %q", got)
	}
}
