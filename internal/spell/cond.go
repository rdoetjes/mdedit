package spell

import (
	"strings"
	"unicode/utf8"
)

// condition is the condition of an affix rule: what the end (for a suffix)
// or the start (for a prefix) of a word must look like for the rule to
// apply. It is a simple pattern: characters, "." for any character, and
// [abc] or [^abc] for one of (or none of) a set.
type condition []condUnit

// condUnit matches one character.
type condUnit struct {
	any   bool
	neg   bool
	runes string
}

func (u condUnit) match(r rune) bool {
	if u.any {
		return true
	}
	return strings.ContainsRune(u.runes, r) != u.neg
}

func parseCondition(s string) condition {
	if s == "." || s == "" {
		return nil
	}
	var c condition
	for s != "" {
		var u condUnit
		u, s = nextCondUnit(s)
		c = append(c, u)
	}
	return c
}

// nextCondUnit parses the unit at the start of s and returns the rest.
func nextCondUnit(s string) (condUnit, string) {
	switch {
	case s[0] == '.':
		return condUnit{any: true}, s[1:]
	case s[0] == '[':
		end := strings.IndexByte(s, ']')
		if end < 0 {
			end = len(s)
		}
		set := s[1:end]
		neg := strings.HasPrefix(set, "^")
		return condUnit{neg: neg, runes: strings.TrimPrefix(set, "^")}, s[min(end+1, len(s)):]
	}
	r, n := utf8.DecodeRuneInString(s)
	return condUnit{runes: string(r)}, s[n:]
}

// matchEnd reports whether the end of word matches.
func (c condition) matchEnd(word string) bool {
	for i := len(c) - 1; i >= 0; i-- {
		r, n := utf8.DecodeLastRuneInString(word)
		if n == 0 || !c[i].match(r) {
			return false
		}
		word = word[:len(word)-n]
	}
	return true
}

// matchStart reports whether the start of word matches.
func (c condition) matchStart(word string) bool {
	for _, u := range c {
		r, n := utf8.DecodeRuneInString(word)
		if n == 0 || !u.match(r) {
			return false
		}
		word = word[n:]
	}
	return true
}
