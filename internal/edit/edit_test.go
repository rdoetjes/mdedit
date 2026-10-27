package edit

import (
	"strings"
	"testing"
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
