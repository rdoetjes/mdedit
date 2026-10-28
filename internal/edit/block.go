package edit

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	listRe  = regexp.MustCompile(`^(\s*)([-*+]|\d+[.)])(\s+)(\[[ xX]\]\s+)?`)
	quoteRe = regexp.MustCompile(`^(\s*)>\s?`)
	headRe  = regexp.MustCompile(`^#{1,6}\s+`)
)

// ListKind selects a list style.
type ListKind int

const (
	Bullets ListKind = iota
	Numbers
	Tasks
)

// lineMapper rewrites one line of a selection.
type lineMapper interface {
	mapLine(line string) string
}

// mapLines rewrites each line of the selected line range. The selection is
// kept on the same text where possible.
func (s State) mapLines(m lineMapper) State {
	from, to := s.selectedLines()
	lines := strings.Split(string(s.Text[from:to]), "\n")
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = m.mapLine(l)
	}
	joined := strings.Join(out, "\n")
	if len(lines) == 1 && s.Start == s.End {
		// Single line: keep the caret on the same text.
		delta := len([]rune(out[0])) - len([]rune(lines[0]))
		caret := min(max(from, s.Start+delta), from+len([]rune(out[0])))
		return s.replace(from, to, joined, caret, caret)
	}
	return s.replace(from, to, joined, from, from+len([]rune(joined)))
}

// selectedText is the text of the whole lines touched by the selection.
func (s State) selectedText() string {
	from, to := s.selectedLines()
	return string(s.Text[from:to])
}

// multiLine reports whether the selection touches more than one line.
func (s State) multiLine() bool {
	return strings.ContainsRune(s.selectedText(), '\n')
}

// ---- Lists ----

// ToggleList turns the selected lines into a list of the given kind, or back
// into plain lines if they already are one.
func ToggleList(s State, kind ListKind) State {
	m := &listMapper{kind: kind, multi: s.multiLine()}
	m.remove = allListOf(strings.Split(s.selectedText(), "\n"), kind, m.multi)
	return s.mapLines(m)
}

// allListOf reports whether every (non-blank, in a multi-line selection)
// line is a list item of the given kind.
func allListOf(lines []string, kind ListKind, multi bool) bool {
	for _, l := range lines {
		if strings.TrimSpace(l) == "" && multi {
			continue
		}
		m := listRe.FindStringSubmatch(l)
		if m == nil || listKind(m) != kind {
			return false
		}
	}
	return true
}

func listKind(m []string) ListKind {
	switch {
	case m[4] != "":
		return Tasks
	case m[2][0] >= '0' && m[2][0] <= '9':
		return Numbers
	}
	return Bullets
}

// listMapper adds or removes list markers, numbering ordered lists.
type listMapper struct {
	kind   ListKind
	multi  bool
	remove bool
	n      int
}

func (m *listMapper) mapLine(l string) string {
	if strings.TrimSpace(l) == "" && m.multi {
		return l
	}
	indent, body := splitMarker(l)
	if m.remove {
		return indent + body
	}
	m.n++
	return indent + m.marker() + body
}

func (m *listMapper) marker() string {
	switch m.kind {
	case Numbers:
		return strconv.Itoa(m.n) + ". "
	case Tasks:
		return "- [ ] "
	}
	return "- "
}

// splitMarker splits a line into its indentation and its text without any
// list marker.
func splitMarker(l string) (indent, body string) {
	if m := listRe.FindStringSubmatch(l); m != nil {
		return m[1], l[len(m[0]):]
	}
	body = strings.TrimLeftFunc(l, unicode.IsSpace)
	return l[:len(l)-len(body)], body
}

// ---- Quotes and headings ----

// ToggleQuote adds a level of block quote to the selected lines, or removes
// one if all of them are quoted.
func ToggleQuote(s State) State {
	m := quoteMapper{remove: true, multi: s.multiLine()}
	for _, l := range strings.Split(s.selectedText(), "\n") {
		if !quoteRe.MatchString(l) {
			m.remove = false
		}
	}
	return s.mapLines(m)
}

type quoteMapper struct{ remove, multi bool }

func (m quoteMapper) mapLine(l string) string {
	if m.remove {
		return quoteRe.ReplaceAllString(l, "$1")
	}
	if l == "" && m.multi {
		return ">" // blank line inside a quoted selection
	}
	return "> " + l
}

// SetHeading makes the selected lines headings of the given level (1-6), or
// plain paragraphs for level 0. Applying a line's current level removes it.
func SetHeading(s State, level int) State {
	first, _, _ := strings.Cut(s.selectedText(), "\n")
	if m := headRe.FindString(first); level > 0 && strings.TrimSpace(m) == strings.Repeat("#", level) {
		level = 0
	}
	return s.mapLines(headingMapper{level: level, single: !s.multiLine()})
}

type headingMapper struct {
	level  int
	single bool
}

func (m headingMapper) mapLine(l string) string {
	l = headRe.ReplaceAllString(l, "")
	// Blank lines inside a multi-line selection stay blank, but a single
	// empty line gets the marker so the heading can be typed after it.
	if m.level == 0 || (strings.TrimSpace(l) == "" && !m.single) {
		return l
	}
	return strings.Repeat("#", m.level) + " " + l
}

// ---- Indentation ----

// Tab handles the Tab key: on a list item or a selection spanning several
// lines it indents the lines (nesting list items); otherwise it inserts
// spaces up to the next tab stop.
func Tab(s State) State {
	if s.multiLine() || listRe.MatchString(s.selectedText()) {
		return s.mapLines(indentMapper{})
	}
	if s.Start != s.End {
		s = s.replace(s.Start, s.End, "", s.Start, s.Start)
	}
	col := s.Start - s.lineStart(s.Start)
	n := TabWidth - col%TabWidth
	return s.replace(s.Start, s.Start, strings.Repeat(" ", n), s.Start+n, s.Start+n)
}

type indentMapper struct{}

func (indentMapper) mapLine(l string) string {
	if strings.TrimSpace(l) == "" {
		return l
	}
	return strings.Repeat(" ", TabWidth) + l
}

// Outdent handles Shift+Tab: it removes up to one tab stop of leading spaces
// from the selected lines.
func Outdent(s State) State {
	return s.mapLines(outdentMapper{})
}

type outdentMapper struct{}

func (outdentMapper) mapLine(l string) string {
	n := 0
	for n < TabWidth && n < len(l) && l[n] == ' ' {
		n++
	}
	return l[n:]
}
