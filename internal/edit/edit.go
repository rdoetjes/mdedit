// Package edit implements the Markdown formatting commands of the editor as
// pure functions on text and a selection, so they can be tested without a UI.
//
// Offsets are rune offsets, as used by Gio's editor.
package edit

import (
	"strconv"
	"strings"
	"unicode"
)

// State is a text with a selection. Start <= End; they are equal when there
// is just a caret.
type State struct {
	Text       []rune
	Start, End int
}

func New(text string, start, end int) State {
	if start > end {
		start, end = end, start
	}
	return State{Text: []rune(text), Start: start, End: end}
}

func (s State) String() string { return string(s.Text) }

// replace substitutes text[from:to] with r and returns the new state with the
// selection set to [selStart, selEnd] (offsets in the new text).
func (s State) replace(from, to int, r string, selStart, selEnd int) State {
	rr := []rune(r)
	t := make([]rune, 0, len(s.Text)-(to-from)+len(rr))
	t = append(t, s.Text[:from]...)
	t = append(t, rr...)
	t = append(t, s.Text[to:]...)
	return State{Text: t, Start: selStart, End: selEnd}
}

// lineStart returns the offset of the start of the line containing pos.
func (s State) lineStart(pos int) int {
	for pos > 0 && s.Text[pos-1] != '\n' {
		pos--
	}
	return pos
}

// lineEnd returns the offset of the newline ending the line containing pos,
// or len(Text).
func (s State) lineEnd(pos int) int {
	for pos < len(s.Text) && s.Text[pos] != '\n' {
		pos++
	}
	return pos
}

// selectedLines returns the range of whole lines touched by the selection.
func (s State) selectedLines() (from, to int) {
	end := s.End
	if end > s.Start && s.Text[end-1] == '\n' {
		end-- // a selection ending at a line start doesn't include that line
	}
	return s.lineStart(s.Start), s.lineEnd(end)
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '\''
}

// ---- Inline formatting ----

// Inline markers.
var (
	Bold      = [2]string{"**", "**"}
	Italic    = [2]string{"*", "*"}
	Underline = [2]string{"<u>", "</u>"}
	Strike    = [2]string{"~~", "~~"}
	Code      = [2]string{"`", "`"}
)

// ToggleInline wraps the selection in the marker pair, or removes the markers
// if the selection is already wrapped. Without a selection it applies to the
// word at the caret, or inserts an empty pair with the caret between.
func ToggleInline(s State, m [2]string) State {
	start, end := s.Start, s.End
	if start == end {
		for start > 0 && isWordRune(s.Text[start-1]) {
			start--
		}
		for end < len(s.Text) && isWordRune(s.Text[end]) {
			end++
		}
		if start == end {
			return s.replace(start, end, m[0]+m[1], start+len([]rune(m[0])), start+len([]rune(m[0])))
		}
	}
	// Leave surrounding whitespace outside the markers.
	for start < end && unicode.IsSpace(s.Text[start]) {
		start++
	}
	for end > start && unicode.IsSpace(s.Text[end-1]) {
		end--
	}
	open, close := []rune(m[0]), []rune(m[1])
	lo, lc := len(open), len(close)

	// Markers just outside the selection.
	if start >= lo && end+lc <= len(s.Text) &&
		hasMarker(s.Text, start-lo, open) && hasMarker(s.Text, end, close) {
		t := s.replace(end, end+lc, "", 0, 0)
		t = t.replace(start-lo, start, "", start-lo, end-lo)
		return t
	}
	// Markers just inside the selection.
	if end-start >= lo+lc && hasMarker(s.Text, start, open) && hasMarker(s.Text, end-lc, close) {
		inner := string(s.Text[start+lo : end-lc])
		return s.replace(start, end, inner, start, end-lo-lc)
	}
	inner := string(s.Text[start:end])
	return s.replace(start, end, m[0]+inner+m[1], start+lo, end+lo)
}

// hasMarker reports whether marker occurs at pos. For markers made of a
// repeated character ("*", "**", "~~") the whole run of that character must
// match, so the "*" of "**bold**" isn't mistaken for italics. A run of three
// counts as both bold and italic.
func hasMarker(t []rune, pos int, marker []rune) bool {
	if pos < 0 || pos+len(marker) > len(t) || string(t[pos:pos+len(marker)]) != string(marker) {
		return false
	}
	c := marker[0]
	if strings.Count(string(marker), string(c)) != len(marker) || (c != '*' && c != '_' && c != '~') {
		return true
	}
	// Measure the full run containing the marker.
	a, b := pos, pos+len(marker)
	for a > 0 && t[a-1] == c {
		a--
	}
	for b < len(t) && t[b] == c {
		b++
	}
	run := b - a
	return run == len(marker) || (run == 3 && c == '*')
}

// ---- Insertions ----

// Newline handles the Enter key. Inside a list item or block quote it starts
// a new item with the same marker (numbers are incremented); on an empty item
// it ends the list instead, leaving a blank line.
func Newline(s State) State {
	if s.Start != s.End {
		s = s.replace(s.Start, s.End, "", s.Start, s.Start)
	}
	pos := s.Start
	ls, le := s.lineStart(pos), s.lineEnd(pos)
	line := string(s.Text[ls:le])
	plain := s.replace(pos, pos, "\n", pos+1, pos+1)

	prefix := ""
	if m := listRe.FindStringSubmatch(line); m != nil {
		if pos-ls < len([]rune(m[0])) {
			return plain
		}
		if strings.TrimSpace(line[len(m[0]):]) == "" {
			return endBlock(s, ls, le)
		}
		marker := m[2]
		if n, err := strconv.Atoi(marker[:len(marker)-1]); err == nil {
			marker = strconv.Itoa(n+1) + marker[len(marker)-1:]
		}
		prefix = m[1] + marker + m[3]
		if m[4] != "" {
			prefix += "[ ] "
		}
	} else if m := quoteRe.FindString(line); m != "" {
		if pos-ls < len([]rune(m)) {
			return plain
		}
		if strings.TrimSpace(line[len(m):]) == "" {
			return endBlock(s, ls, le)
		}
		prefix = strings.TrimRight(m, " ") + " "
	} else {
		return plain
	}
	ins := "\n" + prefix
	n := len([]rune(ins))
	return s.replace(pos, pos, ins, pos+n, pos+n)
}

// endBlock replaces an empty list item or quote line with a blank line and
// puts the caret on a fresh line below it, ending the list.
func endBlock(s State, ls, le int) State {
	return s.replace(ls, le, "\n", ls+1, ls+1)
}

// ownLine inserts block text on its own line(s) after the current line,
// separated from surrounding text by blank lines. caret is an offset into
// block where the caret goes, or -1 for after it.
func ownLine(s State, block string, selFrom, selTo int) State {
	ls, le := s.lineStart(s.Start), s.lineEnd(s.End)
	cur := strings.TrimSpace(string(s.Text[ls:le]))
	var at int
	var before string
	if cur == "" {
		at = ls
		if ls >= 2 && s.Text[ls-2] != '\n' {
			before = "\n"
		}
		// Replace the empty line itself.
		s = s.replace(ls, le, "", ls, ls)
	} else {
		at = le
		before = "\n\n"
	}
	after := "\n"
	rest := s.Text[at:]
	if len(rest) == 0 {
		after = "\n"
	} else if len(rest) >= 2 && rest[0] == '\n' && rest[1] == '\n' {
		after = ""
	} else if rest[0] == '\n' {
		after = "\n"
	} else {
		after = "\n\n"
	}
	ins := before + block + after
	base := at + len([]rune(before))
	return s.replace(at, at, ins, base+selFrom, base+selTo)
}

// Table inserts an empty table with the given number of columns and body
// rows, with the caret in the first header cell.
func Table(s State, cols, rows int) State {
	cols = max(cols, 1)
	rows = max(rows, 0)
	row := strings.Repeat("|     ", cols) + "|"
	sep := strings.Repeat("| --- ", cols) + "|"
	lines := []string{row, sep}
	for i := 0; i < rows; i++ {
		lines = append(lines, row)
	}
	return ownLine(s, strings.Join(lines, "\n"), 2, 2)
}

// CodeBlock fences the selected lines, or inserts an empty fenced block.
func CodeBlock(s State) State {
	if s.Start != s.End {
		from, to := s.selectedLines()
		body := string(s.Text[from:to])
		block := "```\n" + body + "\n```"
		return s.replace(from, to, block, from+3, from+3)
	}
	return ownLine(s, "```\n\n```", 3, 3)
}

// Rule inserts a horizontal rule.
func Rule(s State) State {
	n := len([]rune("---"))
	return ownLine(s, "---", n, n)
}

// Link turns the selection into a link and selects the part to fill in.
func Link(s State, image bool) State {
	sel := string(s.Text[s.Start:s.End])
	bang := ""
	if image {
		bang = "!"
	}
	b := len([]rune(bang))
	if strings.Contains(sel, "://") && !strings.ContainsAny(sel, " \n") {
		txt := bang + "[](" + sel + ")"
		return s.replace(s.Start, s.End, txt, s.Start+b+1, s.Start+b+1)
	}
	label := sel
	if label == "" {
		label = "text"
		if image {
			label = "alt text"
		}
	}
	url := "https://"
	if image {
		url = "image.png"
	}
	txt := bang + "[" + label + "](" + url + ")"
	urlStart := s.Start + b + 1 + len([]rune(label)) + 2
	if sel == "" {
		return s.replace(s.Start, s.End, txt, s.Start+b+1, s.Start+b+1+len([]rune(label)))
	}
	return s.replace(s.Start, s.End, txt, urlStart, urlStart+len([]rune(url)))
}

// ---- Tabs and indentation ----

// TabWidth is the tab stop distance; CommonMark uses 4.
const TabWidth = 4

// ExpandTabs replaces tab characters with spaces up to the next tab stop,
// which is how CommonMark interprets them. The selection follows the text.
func ExpandTabs(s State) State {
	var out []rune
	start, end := s.Start, s.End
	col := 0
	for i, r := range s.Text {
		if i == s.Start {
			start = len(out)
		}
		if i == s.End {
			end = len(out)
		}
		switch r {
		case '\t':
			n := TabWidth - col%TabWidth
			for j := 0; j < n; j++ {
				out = append(out, ' ')
			}
			col += n
		case '\n':
			out = append(out, r)
			col = 0
		default:
			out = append(out, r)
			col++
		}
	}
	if s.Start == len(s.Text) {
		start = len(out)
	}
	if s.End == len(s.Text) {
		end = len(out)
	}
	return State{Text: out, Start: start, End: end}
}

// InlineMath is the marker pair for inline maths.
var InlineMath = [2]string{"$", "$"}

// MathBlock wraps the selected lines in $$ … $$, or inserts an empty maths
// block with the caret inside.
func MathBlock(s State) State {
	if s.Start != s.End {
		from, to := s.selectedLines()
		body := string(s.Text[from:to])
		block := "$$\n" + body + "\n$$"
		return s.replace(from, to, block, from+3, from+3+len([]rune(body)))
	}
	return ownLine(s, "$$\n\n$$", 3, 3)
}
