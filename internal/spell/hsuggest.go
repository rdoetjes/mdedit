package spell

import (
	"slices"
	"strings"
	"unicode"
)

// Suggest returns up to n corrections for word, the most likely first.
// Candidates come from the REP table of common mistakes, from single
// edits (with the letters of TRY) checked against the dictionary, and from
// stems within two edits.
func (h *Hunspell) Suggest(word string, n int) []string {
	lower := strings.ToLower(h.convert(normalise(word)))
	target := []rune(lower)
	if len(target) == 0 {
		return nil
	}
	found := h.stems.candidates(nil, target)
	for _, w := range h.repCandidates(lower) {
		found = append(found, candidate{w, scoreWord(target, w) - costEdit})
	}
	for _, w := range h.editCandidates(target) {
		found = append(found, candidate{w, scoreWord(target, w)})
	}
	slices.SortFunc(penaliseCase(found, word), byScore)
	return matchCase(word, found, n)
}

// repCandidates applies each REP pattern at each place it matches.
func (h *Hunspell) repCandidates(word string) []string {
	var out []string
	for _, r := range h.rep {
		for i := 0; i+len(r.from) <= len(word); i++ {
			if !r.matchesAt(word, i) {
				continue
			}
			c := word[:i] + r.to + word[i+len(r.from):]
			if h.Correct(c) {
				out = append(out, c)
			}
		}
	}
	return out
}

func (r repPattern) matchesAt(word string, i int) bool {
	if r.start && i != 0 || r.end && i+len(r.from) != len(word) {
		return false
	}
	return strings.HasPrefix(word[i:], r.from)
}

// editCandidates returns the correct words one edit away from target:
// a letter deleted, swapped with the next, replaced or inserted.
func (h *Hunspell) editCandidates(target []rune) []string {
	var out []string
	letters := h.letters()
	for i := 0; i <= len(target); i++ {
		out = h.keep(out, edited(target, i, i+1, nil))
		if i+1 < len(target) {
			out = h.keep(out, edited(target, i, i+2, []rune{target[i+1], target[i]}))
		}
		for _, r := range letters {
			out = h.keep(out, edited(target, i, i+1, []rune{r}))
			out = h.keep(out, edited(target, i, i, []rune{r}))
		}
	}
	return out
}

// edited returns target with [from, to) replaced by with, or "" if to is
// past the end.
func edited(target []rune, from, to int, with []rune) string {
	if to > len(target) {
		return ""
	}
	out := make([]rune, 0, len(target)+1)
	out = append(append(append(out, target[:from]...), with...), target[to:]...)
	return string(out)
}

// keep appends w if it is a correct word.
func (h *Hunspell) keep(out []string, w string) []string {
	if w != "" && h.Correct(w) {
		return append(out, w)
	}
	return out
}

// letters are the lower-case characters tried in edits: those of TRY, or
// else a-z. Upper-case ones would only produce oddly cased duplicates.
func (h *Hunspell) letters() []rune {
	var out []rune
	for _, r := range h.try {
		if !unicode.IsUpper(r) {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return []rune("abcdefghijklmnopqrstuvwxyz")
	}
	return out
}
