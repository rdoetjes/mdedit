package ui

import (
	"slices"
	"unicode"

	"mdedit/internal/mdmodel"
)

// SourceClick is a click in the preview, described so it can be found in
// the Markdown source: on the Nth occurrence (from 0) of Word in the
// rendered text of block Block, Offset characters into it. Word is empty
// when the click wasn't on a word (maths, an image, empty space).
type SourceClick struct {
	Block  int
	Word   string
	Nth    int
	Offset int
}

// showSource puts the editor's caret on what was clicked in the preview,
// on the editor's top line, and gives the editor the keyboard focus, so
// typing fixes it there. The preview stays where it is.
func (a *App) showSource(c SourceClick) {
	if c.Block >= len(a.doc.Blocks) {
		return
	}
	a.showAtTop(sourceOffset([]rune(a.editor.Text()), a.doc, c))
}

// sourceOffset returns the rune offset in text that a preview click
// refers to: the same occurrence of the word in the block's source, or
// else the start of the block.
func sourceOffset(text []rune, doc *mdmodel.Document, c SourceClick) int {
	from, to := blockRange(text, doc, c.Block)
	if c.Word == "" {
		return from
	}
	word := []rune(c.Word)
	starts := wordOccurrences(text[from:to], word)
	if len(starts) == 0 {
		return from
	}
	at := starts[min(c.Nth, len(starts)-1)]
	return from + at + min(max(c.Offset, 0), len(word))
}

// blockRange returns the rune offsets of the source lines of block i:
// from the line it starts on to the line the next block starts on.
func blockRange(text []rune, doc *mdmodel.Document, i int) (int, int) {
	starts := lineStarts(text)
	from := starts[min(doc.Blocks[i].Line, len(starts)-1)]
	to := len(text)
	if i+1 < len(doc.Blocks) && doc.Blocks[i+1].Line < len(starts) {
		to = max(starts[doc.Blocks[i+1].Line], from)
	}
	return from, to
}

// wordOccurrences returns where word occurs in text as a whole word.
func wordOccurrences(text, word []rune) []int {
	var out []int
	for i := 0; i+len(word) <= len(text); i++ {
		if slices.Equal(text[i:i+len(word)], word) && wholeWord(text, i, i+len(word)) {
			out = append(out, i)
		}
	}
	return out
}

func wholeWord(text []rune, from, to int) bool {
	return (from == 0 || !isWordRune(text[from-1])) && (to == len(text) || !isWordRune(text[to]))
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// wordAt returns the word in s at or nearest after index i (or else
// before it), as rune offsets; ok is false if s has no word.
func wordAt(s []rune, i int) (from, to int, ok bool) {
	at, found := wordRuneNear(s, min(max(i, 0), len(s)))
	if !found {
		return 0, 0, false
	}
	from, to = at, at+1
	for from > 0 && isWordRune(s[from-1]) {
		from--
	}
	for to < len(s) && isWordRune(s[to]) {
		to++
	}
	return from, to, true
}

// wordRuneNear returns the first letter or digit at or after i, or else
// the last one before it.
func wordRuneNear(s []rune, i int) (int, bool) {
	for j := i; j < len(s); j++ {
		if isWordRune(s[j]) {
			return j, true
		}
	}
	for j := i - 1; j >= 0; j-- {
		if isWordRune(s[j]) {
			return j, true
		}
	}
	return 0, false
}

// countWord counts the whole-word occurrences of word in s.
func countWord(s, word []rune) int {
	return len(wordOccurrences(s, word))
}
