package edit

import (
	"strings"

	"mdedit/internal/mdmodel"
)

// Table of contents markers, as used by markdown-toc and VS Code's Markdown
// All in One. They are HTML comments, so renderers don't show them.
const (
	TOCStart = "<!-- toc -->"
	TOCEnd   = "<!-- /toc -->"
)

// BuildTOC returns a nested list of links to the document's headings, using
// the anchors the parser assigns. Headings inside quotes or lists are left
// out.
func BuildTOC(text string) string {
	doc := mdmodel.Parse([]byte(text))
	type entry struct {
		level     int
		label, id string
	}
	var entries []entry
	minLevel := 7
	for _, b := range doc.Blocks {
		if b.Kind != mdmodel.Heading || len(b.Decor) > 0 || b.ID == "" {
			continue
		}
		var label strings.Builder
		for _, s := range b.Spans {
			label.WriteString(s.Text)
		}
		l := strings.TrimSpace(label.String())
		if l == "" {
			continue
		}
		entries = append(entries, entry{b.Level, l, b.ID})
		minLevel = min(minLevel, b.Level)
	}
	var out []string
	depth := -1
	for _, e := range entries {
		// Nest at most one level deeper than the previous entry, so skipped
		// heading levels don't produce over-indented (code block) lines.
		depth = min(e.level-minLevel, depth+1)
		label := strings.NewReplacer(`\`, `\\`, "[", `\[`, "]", `\]`).Replace(e.label)
		out = append(out, strings.Repeat("  ", depth)+"- ["+label+"](#"+e.id+")")
	}
	return strings.Join(out, "\n")
}

func tocBlock(text string) string {
	toc := BuildTOC(text)
	if toc == "" {
		toc = "_No headings yet._"
	}
	return TOCStart + "\n\n" + toc + "\n\n" + TOCEnd
}

// FindTOC returns the rune range of the table of contents, from the start of
// its opening marker to the end of its closing marker.
func FindTOC(s State) (from, to int, ok bool) {
	t := string(s.Text)
	i := strings.Index(t, TOCStart)
	if i < 0 {
		return 0, 0, false
	}
	j := strings.Index(t[i:], TOCEnd)
	if j < 0 {
		return 0, 0, false
	}
	from = len([]rune(t[:i]))
	to = len([]rune(t[:i+j+len(TOCEnd)]))
	return from, to, true
}

// InsertTOC inserts a table of contents at the caret, or updates the
// existing one, since a document has only one.
func InsertTOC(s State) State {
	if _, _, ok := FindTOC(s); ok {
		return UpdateTOC(s)
	}
	block := tocBlock(string(s.Text))
	n := len([]rune(block))
	return ownLine(s, block, n, n)
}

// UpdateTOC rewrites the existing table of contents from the current
// headings. The selection stays on the same text.
func UpdateTOC(s State) State {
	from, to, ok := FindTOC(s)
	if !ok {
		return s
	}
	block := tocBlock(string(s.Text))
	delta := len([]rune(block)) - (to - from)
	adjust := func(p int) int {
		switch {
		case p >= to:
			return p + delta
		case p > from:
			return from
		}
		return p
	}
	return s.replace(from, to, block, adjust(s.Start), adjust(s.End))
}
