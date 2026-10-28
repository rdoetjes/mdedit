// Package spell checks the spelling of the prose in a Markdown document and
// suggests corrections.
//
// The English word list is SCOWL/ESDB size 60 with American and British
// (-ise) spellings, from https://app.aspell.net/create (see
// SCOWL-LICENSE.txt). en.txt.gz holds one word per line, prefixed by a digit
// that says how common the word is: 0 for the most common words (SCOWL size
// 35), then 1, 2 and 3 for sizes 40, 50 and 60. Suggestions prefer common
// words.
package spell

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

//go:embed en.txt.gz
var englishData []byte

// Dictionary is a set of correctly spelt words.
type Dictionary struct {
	words map[string]uint8 // word → commonness rank, 0 most common
	// byLen holds the words by their length in runes, for suggestions.
	byLen [][]entry
}

// entry is a word as compared when looking for suggestions.
type entry struct {
	word  string
	lower []rune
	rank  uint8
}

var (
	englishOnce sync.Once
	english     *Dictionary
)

// English returns the built-in English dictionary, loading it on first use.
// Words added to it are shared by all its users.
func English() *Dictionary {
	englishOnce.Do(loadEnglish)
	return english
}

func loadEnglish() {
	english = &Dictionary{words: make(map[string]uint8, 120000)}
	zr, err := gzip.NewReader(bytes.NewReader(englishData))
	if err != nil {
		panic("spell: broken word list: " + err.Error())
	}
	sc := bufio.NewScanner(zr)
	for sc.Scan() {
		line := sc.Text()
		if len(line) > 1 {
			english.add(line[1:], line[0]-'0')
		}
	}
}

// New returns a dictionary holding just the given words.
func New(words ...string) *Dictionary {
	d := &Dictionary{words: make(map[string]uint8)}
	for _, w := range words {
		d.Add(w)
	}
	return d
}

// Add accepts word as correctly spelt from now on.
func (d *Dictionary) Add(word string) {
	word = normalise(word)
	if word != "" {
		d.add(word, 0)
	}
}

func (d *Dictionary) add(word string, rank uint8) {
	if _, ok := d.words[word]; ok {
		return
	}
	d.words[word] = rank
	e := entry{word: word, lower: []rune(strings.ToLower(word)), rank: rank}
	for len(d.byLen) <= len(e.lower) {
		d.byLen = append(d.byLen, nil)
	}
	d.byLen[len(e.lower)] = append(d.byLen[len(e.lower)], e)
}

// Correct reports whether word is spelt correctly. A capitalised or
// upper-case word is also correct when its lower-case form is, and a
// possessive when the word without 's is.
func (d *Dictionary) Correct(word string) bool {
	word = normalise(word)
	if d.known(word) {
		return true
	}
	if base, ok := strings.CutSuffix(word, "'s"); ok && d.known(base) {
		return true
	}
	if base, ok := strings.CutSuffix(word, "s'"); ok && d.known(base+"s") {
		return true
	}
	return false
}

// known looks the word up as written, in lower case, and capitalised.
func (d *Dictionary) known(word string) bool {
	if _, ok := d.words[word]; ok {
		return true
	}
	lower := strings.ToLower(word)
	if _, ok := d.words[lower]; ok {
		return true
	}
	if isUpper(word) {
		_, ok := d.words[capitalise(lower)]
		return ok
	}
	return false
}

// normalise turns typographic apostrophes into straight ones.
func normalise(word string) string {
	return strings.ReplaceAll(word, "’", "'")
}

// isUpper reports whether word has no lower-case letters.
func isUpper(word string) bool {
	return strings.ToUpper(word) == word
}

// isCapitalised reports whether word starts with an upper-case letter
// followed only by lower-case ones.
func isCapitalised(word string) bool {
	r, n := utf8.DecodeRuneInString(word)
	return unicode.IsUpper(r) && strings.ToLower(word[n:]) == word[n:]
}

func capitalise(word string) string {
	r, n := utf8.DecodeRuneInString(word)
	return string(unicode.ToUpper(r)) + word[n:]
}
