package edit

import "unicode"

// Query is a search: the text to find and whether case must match.
type Query struct {
	Text      string
	MatchCase bool
}

// FindAll returns the rune offsets of the non-overlapping matches of q in
// text, at most limit of them (0 means no limit).
func FindAll(text []rune, q Query, limit int) []int {
	pat := []rune(q.Text)
	if len(pat) == 0 {
		return nil
	}
	var out []int
	for i := 0; i+len(pat) <= len(text); i++ {
		if !q.matchAt(text, i, pat) {
			continue
		}
		out = append(out, i)
		if limit > 0 && len(out) == limit {
			break
		}
		i += len(pat) - 1
	}
	return out
}

// matchAt reports whether pat occurs in text at i. Case-insensitive
// matching folds rune by rune, so offsets stay the same.
func (q Query) matchAt(text []rune, i int, pat []rune) bool {
	for j, p := range pat {
		if !q.sameRune(text[i+j], p) {
			return false
		}
	}
	return true
}

func (q Query) sameRune(a, b rune) bool {
	if q.MatchCase {
		return a == b
	}
	return a == b || unicode.ToLower(a) == unicode.ToLower(b)
}

// Matches reports whether the selection of s is exactly a match of q.
func (q Query) Matches(s State) bool {
	pat := []rune(q.Text)
	return len(pat) > 0 && s.End-s.Start == len(pat) && q.matchAt(s.Text, s.Start, pat)
}

// NextMatch returns the index of the first match at or after pos, wrapping
// around to the first match, or -1 if there are none.
func NextMatch(matches []int, pos int) int {
	for i, m := range matches {
		if m >= pos {
			return i
		}
	}
	if len(matches) > 0 {
		return 0
	}
	return -1
}

// PrevMatch returns the index of the last match before pos, wrapping around
// to the last match, or -1 if there are none.
func PrevMatch(matches []int, pos int) int {
	for i := len(matches) - 1; i >= 0; i-- {
		if matches[i] < pos {
			return i
		}
	}
	return len(matches) - 1
}

// ReplaceSelection replaces the selection with repl if it is a match of q,
// and selects the replacement. It reports whether it replaced anything.
func ReplaceSelection(s State, q Query, repl string) (State, bool) {
	if !q.Matches(s) {
		return s, false
	}
	end := s.Start + len([]rune(repl))
	return s.replace(s.Start, s.End, repl, s.Start, end), true
}

// ReplaceAll replaces every match of q with repl and returns the new state
// and the number of replacements. The caret moves to the start.
func ReplaceAll(s State, q Query, repl string) (State, int) {
	matches := FindAll(s.Text, q, 0)
	if len(matches) == 0 {
		return s, 0
	}
	patLen := len([]rune(q.Text))
	r := []rune(repl)
	out := make([]rune, 0, len(s.Text)+len(matches)*(len(r)-patLen))
	last := 0
	for _, m := range matches {
		out = append(out, s.Text[last:m]...)
		out = append(out, r...)
		last = m + patLen
	}
	out = append(out, s.Text[last:]...)
	return State{Text: out}, len(matches)
}

// ReplaceWord replaces text[start:end] with word and puts the caret after
// it, if that part of the text still reads old; otherwise it changes
// nothing.
func ReplaceWord(s State, start, end int, old, word string) State {
	if start < 0 || start > end || end > len(s.Text) || string(s.Text[start:end]) != old {
		return s
	}
	caret := start + len([]rune(word))
	return s.replace(start, end, word, caret, caret)
}
