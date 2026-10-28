package spell

import (
	"slices"
	"strings"
)

// Edit costs, in quarters of an edit. Slips of the finger cost less than
// other mistakes, so they rank higher among the suggestions.
const (
	costEdit      = 4 // insert, delete or substitute a letter
	costSlip      = 3 // a neighbouring key
	costTwist     = 2 // swapped letters, or a letter typed once instead of twice
	maxCost       = 2 * costEdit
	costFirstRune = 2 // the first letter is rarely the wrong one
)

// candidate is a dictionary word with its distance from the misspelling.
type candidate struct {
	word  string
	score int
}

// Suggest returns up to n corrections for word, the most likely first.
func (d *Dictionary) Suggest(word string, n int) []string {
	target := []rune(strings.ToLower(normalise(word)))
	if len(target) == 0 {
		return nil
	}
	var found []candidate
	var rows [3][]int
	for l := max(len(target)-2, 1); l <= len(target)+2 && l < len(d.byLen); l++ {
		for i := range rows {
			rows[i] = make([]int, l+1)
		}
		found = collect(found, target, d.byLen[l], rows)
	}
	slices.SortFunc(found, byScore)
	return matchCase(word, found, n)
}

// collect appends the entries within reach of target. The entries are all
// of one length; rows is the scratch space for distance.
func collect(found []candidate, target []rune, entries []entry, rows [3][]int) []candidate {
	for i := range entries {
		e := &entries[i]
		cost := distance(target, e.lower, rows)
		if cost > maxCost {
			continue
		}
		score := 2*cost + int(e.rank)
		if e.lower[0] != target[0] {
			score += costFirstRune
		}
		found = append(found, candidate{e.word, score})
	}
	return found
}

func byScore(a, b candidate) int {
	if a.score != b.score {
		return a.score - b.score
	}
	return strings.Compare(a.word, b.word)
}

// matchCase writes the first n distinct candidates the way word is written:
// capitalised or in upper case.
func matchCase(word string, found []candidate, n int) []string {
	var out []string
	for _, c := range found {
		w := c.word
		switch {
		case len([]rune(word)) > 1 && isUpper(word):
			w = strings.ToUpper(w)
		case isCapitalised(word):
			w = capitalise(w)
		}
		if w != word && !slices.Contains(out, w) {
			out = append(out, w)
		}
		if len(out) == n {
			break
		}
	}
	return out
}

// distance is the cost of turning a into b by inserting, deleting and
// substituting letters and swapping neighbouring letters (the optimal string
// alignment distance), or more than maxCost if it is larger than that.
// Row i of the table is rows[i%3], each len(b)+1 long; a swap looks two
// rows back.
func distance(a, b []rune, rows [3][]int) int {
	for j := range rows[0] {
		rows[0][j] = j * costEdit
	}
	for i := 1; i <= len(a); i++ {
		if fillRow(a, b, i, rows[(i+1)%3], rows[(i-1)%3], rows[i%3]) > maxCost {
			return maxCost + 1
		}
	}
	return rows[len(a)%3][len(b)]
}

// fillRow computes row i of the distance table and returns its smallest
// value.
func fillRow(a, b []rune, i int, prev2, prev, cur []int) int {
	cur[0] = i * costEdit
	least := cur[0]
	for j := 1; j <= len(b); j++ {
		c := min(prev[j]+indelCost(a, i), cur[j-1]+indelCost(b, j), prev[j-1]+substCost(a[i-1], b[j-1]))
		if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
			c = min(c, prev2[j-2]+costTwist)
		}
		cur[j] = c
		least = min(least, c)
	}
	return least
}

// indelCost is the cost of inserting or deleting s[i-1]: less if it
// doubles the letter before it.
func indelCost(s []rune, i int) int {
	if i > 1 && s[i-1] == s[i-2] {
		return costTwist
	}
	return costEdit
}

func substCost(x, y rune) int {
	switch {
	case x == y:
		return 0
	case x >= 'a' && x <= 'z' && y >= 'a' && y <= 'z' && adjacent[x-'a'][y-'a']:
		return costSlip
	}
	return costEdit
}

// keyboard is the QWERTY layout, for finding neighbouring keys.
var keyboard = [3]string{"qwertyuiop", "asdfghjkl", "zxcvbnm"}

// adjacent[x-'a'][y-'a'] tells whether keys x and y are neighbours.
var adjacent = adjacentKeys()

func adjacentKeys() (t [26][26]bool) {
	for x := 'a'; x <= 'z'; x++ {
		for y := 'a'; y <= 'z'; y++ {
			t[x-'a'][y-'a'] = neighbours(x, y)
		}
	}
	return t
}

// neighbours reports whether x and y are next to each other on a QWERTY
// keyboard, in the same row or the row above or below.
func neighbours(x, y rune) bool {
	rx, cx := keyAt(x)
	ry, cy := keyAt(y)
	if rx < 0 || ry < 0 {
		return false
	}
	switch rx - ry {
	case 0:
		return cx-cy == 1 || cy-cx == 1
	case 1: // x is a row below y, which is shifted left by about half a key
		return cy == cx || cy == cx+1
	case -1:
		return cx == cy || cx == cy+1
	}
	return false
}

// keyAt returns the row and column of r on the keyboard, or -1, -1.
func keyAt(r rune) (int, int) {
	for row, keys := range keyboard {
		if c := strings.IndexRune(keys, r); c >= 0 {
			return row, c
		}
	}
	return -1, -1
}
