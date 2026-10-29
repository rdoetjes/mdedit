package spell

import (
	"slices"
	"strings"
)

// Speller checks words and suggests corrections. *Dictionary (the
// built-in English word list and the user's own words), *Hunspell
// (downloaded dictionaries) and Set implement it.
type Speller interface {
	Correct(word string) bool
	Suggest(word string, n int) []string
}

// Set combines spellers: a word is correct if any of them accepts it, so
// several languages can be checked at once.
type Set []Speller

func (s Set) Correct(word string) bool {
	for _, sp := range s {
		if sp.Correct(word) {
			return true
		}
	}
	return false
}

// Suggest merges the suggestions of all spellers, ranked by how close
// they are to word; each speller's own order breaks ties.
func (s Set) Suggest(word string, n int) []string {
	target := []rune(strings.ToLower(normalise(word)))
	if len(target) == 0 {
		return nil
	}
	var found []candidate
	for _, sp := range s {
		for i, w := range sp.Suggest(word, n) {
			found = append(found, candidate{w, 4*scoreWord(target, w) + i})
		}
	}
	slices.SortStableFunc(found, byScore)
	var out []string
	for _, c := range found {
		if !slices.Contains(out, c.word) && len(out) < n {
			out = append(out, c.word)
		}
	}
	return out
}
