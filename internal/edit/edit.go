// Package edit implements the Markdown formatting commands of the editor as
// pure functions on text and a selection, so they can be tested without a UI.
//
// Offsets are rune offsets, as used by Gio's editor.
package edit

import (
	"regexp"
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

// ---- Block formatting ----

var (
	listRe  = regexp.MustCompile(`^(\s*)([-*+]|\d+[.)])(\s+)(\[[ xX]\]\s+)?`)
	quoteRe = regexp.MustCompile(`^(\s*)>\s?`)
	headRe  = regexp.MustCompile(`^#{1,6}\s+`)
	alignRe = regexp.MustCompile(`^\s*<(div|p)\s+align\s*=\s*["']?(left|center|right)["']?\s*>\s*$`)
	closeRe = regexp.MustCompile(`^\s*</(div|p)>\s*$`)
)

// ListKind selects a list style.
type ListKind int

const (
	Bullets ListKind = iota
	Numbers
	Tasks
)

// mapLines rewrites each line of the selected line range with f, which gets
// the line index within the range. The selection is kept on the same text
// where possible.
func (s State) mapLines(f func(i int, line string) string) State {
	from, to := s.selectedLines()
	lines := strings.Split(string(s.Text[from:to]), "\n")
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = f(i, l)
	}
	joined := strings.Join(out, "\n")
	if len(lines) == 1 && s.Start == s.End {
		// Single line: keep the caret on the same text.
		delta := len([]rune(out[0])) - len([]rune(lines[0]))
		caret := max(from, s.Start+delta)
		caret = min(caret, from+len([]rune(out[0])))
		return s.replace(from, to, joined, caret, caret)
	}
	return s.replace(from, to, joined, from, from+len([]rune(joined)))
}

// ToggleList turns the selected lines into a list of the given kind, or back
// into plain lines if they already are one.
func ToggleList(s State, kind ListKind) State {
	from, to := s.selectedLines()
	lines := strings.Split(string(s.Text[from:to]), "\n")
	all := true
	for _, l := range lines {
		if strings.TrimSpace(l) == "" && len(lines) > 1 {
			continue
		}
		m := listRe.FindStringSubmatch(l)
		if m == nil || listKind(m) != kind {
			all = false
			break
		}
	}
	n := 0
	return s.mapLines(func(i int, l string) string {
		if strings.TrimSpace(l) == "" && len(lines) > 1 {
			return l
		}
		indent, body := "", l
		if m := listRe.FindStringSubmatch(l); m != nil {
			indent, body = m[1], l[len(m[0]):]
		} else {
			body = strings.TrimLeftFunc(l, unicode.IsSpace)
			indent = l[:len(l)-len(body)]
		}
		if all {
			return indent + body
		}
		n++
		switch kind {
		case Numbers:
			return indent + strconv.Itoa(n) + ". " + body
		case Tasks:
			return indent + "- [ ] " + body
		default:
			return indent + "- " + body
		}
	})
}

func listKind(m []string) ListKind {
	switch {
	case m[4] != "":
		return Tasks
	case m[2][0] >= '0' && m[2][0] <= '9':
		return Numbers
	default:
		return Bullets
	}
}

// ToggleQuote adds a level of block quote to the selected lines, or removes
// one if all of them are quoted.
func ToggleQuote(s State) State {
	from, to := s.selectedLines()
	all := true
	for _, l := range strings.Split(string(s.Text[from:to]), "\n") {
		if !quoteRe.MatchString(l) {
			all = false
		}
	}
	return s.mapLines(func(_ int, l string) string {
		if all {
			return quoteRe.ReplaceAllString(l, "$1")
		}
		if l == "" {
			return ">"
		}
		return "> " + l
	})
}

// SetHeading makes the selected lines headings of the given level (1-6), or
// plain paragraphs for level 0. Applying a line's current level removes it.
func SetHeading(s State, level int) State {
	from, to := s.selectedLines()
	first := strings.SplitN(string(s.Text[from:to]), "\n", 2)[0]
	if m := headRe.FindString(first); level > 0 && strings.TrimSpace(m) == strings.Repeat("#", level) {
		level = 0
	}
	return s.mapLines(func(_ int, l string) string {
		l = headRe.ReplaceAllString(l, "")
		if level == 0 || strings.TrimSpace(l) == "" {
			return l
		}
		return strings.Repeat("#", level) + " " + l
	})
}

// Alignment values for SetAlign.
const (
	AlignLeft   = "left"
	AlignCenter = "center"
	AlignRight  = "right"
)

// SetAlign aligns the paragraph at the selection by wrapping it in
// <div align="...">, the convention GitHub and Typora render. The blank
// lines inside the div keep its content parsed as Markdown. Left alignment
// removes the wrapper.
func SetAlign(s State, align string) State {
	lines := strings.Split(string(s.Text), "\n")
	offsets := make([]int, len(lines)+1)
	for i, l := range lines {
		offsets[i+1] = offsets[i] + len([]rune(l)) + 1
	}
	lineOf := func(pos int) int {
		for i := range lines {
			if pos < offsets[i+1] {
				return i
			}
		}
		return len(lines) - 1
	}
	first, last := lineOf(s.Start), lineOf(max(s.Start, s.End-1))
	if s.End > s.Start && s.End == offsets[last] {
		last--
	}

	// Inside an existing aligned div?
	open, close := -1, -1
	for i := first; i >= 0; i-- {
		if alignRe.MatchString(lines[i]) {
			open = i
			break
		}
		if closeRe.MatchString(lines[i]) && i != first {
			break
		}
	}
	if open >= 0 {
		for i := max(last, open+1); i < len(lines); i++ {
			if closeRe.MatchString(lines[i]) {
				close = i
				break
			}
			if alignRe.MatchString(lines[i]) {
				break
			}
		}
	}
	shift := func(pos, line, by int) int {
		if pos >= offsets[line] {
			return pos + by
		}
		return pos
	}
	if open >= 0 && close >= 0 {
		if align == AlignLeft {
			// Remove the closing line (and a blank line before it), then the
			// opening line (and a blank line after it).
			cFrom, cTo := close, close
			if cFrom-1 > open && strings.TrimSpace(lines[cFrom-1]) == "" {
				cFrom--
			}
			oFrom, oTo := open, open
			if oTo+1 < cFrom && strings.TrimSpace(lines[oTo+1]) == "" {
				oTo++
			}
			t := s.replace(offsets[cFrom]-1, min(offsets[cTo+1]-1, len(s.Text)), "", 0, 0)
			removed := offsets[oTo+1] - offsets[oFrom]
			start, end := s.Start, s.End
			if start >= offsets[oTo+1] {
				start -= removed
				end -= removed
			}
			return t.replace(offsets[oFrom], offsets[oTo+1], "", max(start, offsets[oFrom]), max(end, offsets[oFrom]))
		}
		tag := `<div align="` + align + `">`
		by := len([]rune(tag)) - len([]rune(lines[open]))
		return s.replace(offsets[open], offsets[open]+len([]rune(lines[open])), tag,
			shift(s.Start, open+1, by), shift(s.End, open+1, by))
	}
	if align == AlignLeft {
		return s
	}
	// Extend to the whole paragraph and wrap it.
	for first > 0 && strings.TrimSpace(lines[first-1]) != "" {
		first--
	}
	for last < len(lines)-1 && strings.TrimSpace(lines[last+1]) != "" {
		last++
	}
	before := `<div align="` + align + `">` + "\n\n"
	after := "\n\n</div>"
	endOff := offsets[last] + len([]rune(lines[last]))
	t := s.replace(endOff, endOff, after, 0, 0)
	by := len([]rune(before))
	return t.replace(offsets[first], offsets[first], before, s.Start+by, s.End+by)
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
