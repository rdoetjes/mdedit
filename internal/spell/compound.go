package spell

import "unicode/utf8"

// maxCompoundParts limits how many words a compound may join.
const maxCompoundParts = 6

// part positions in a compound.
const (
	partBegin = iota
	partMiddle
	partEnd
)

// compoundFrom reports whether word[start:] completes a compound of which
// it is part number n (0-based).
func (h *Hunspell) compoundFrom(word string, start, n int) bool {
	pos := partMiddle
	if n == 0 {
		pos = partBegin
	}
	for end := advance(word, start, h.compoundMin); end > 0 && end < len(word); end = advance(word, end, 1) {
		if utf8.RuneCountInString(word[end:]) < h.compoundMin {
			return false
		}
		if !h.partOK(word[start:end], pos) {
			continue
		}
		if h.partOK(word[end:], partEnd) {
			return true
		}
		if n+2 < maxCompoundParts && h.compoundFrom(word, end, n+1) {
			return true
		}
	}
	return false
}

// advance returns the byte offset n characters after i in word, or -1 if
// word ends first.
func advance(word string, i, n int) int {
	for ; n > 0; n-- {
		if i >= len(word) {
			return -1
		}
		_, size := utf8.DecodeRuneInString(word[i:])
		i += size
	}
	return i
}

// partOK reports whether part may stand at pos in a compound.
func (h *Hunspell) partOK(part string, pos int) bool {
	if h.forbiddenWord(part) {
		return false
	}
	for _, f := range h.forms(part) {
		if h.positionAllowed(f, pos) && h.affixesInCompound(f, pos) && h.affixesComplete(f) {
			return true
		}
	}
	return false
}

func (h *Hunspell) positionAllowed(f form, pos int) bool {
	s := h.special
	if f.has(s.compound) {
		return true
	}
	switch pos {
	case partBegin:
		return f.has(s.begin)
	case partMiddle:
		return f.has(s.middle)
	}
	return f.has(s.end)
}

// affixesInCompound: inside a compound, a suffix may only end the last
// part and a prefix only start the first, unless the affix carries
// COMPOUNDPERMITFLAG.
func (h *Hunspell) affixesInCompound(f form, pos int) bool {
	permit := h.special.permit
	if f.pfx != nil && pos != partBegin && !f.pfx.cont.has(permit) {
		return false
	}
	if f.sfx != nil && pos != partEnd && !f.sfx.cont.has(permit) {
		return false
	}
	return true
}

// ruleCompound reports whether word is made of dictionary stems following
// one of the COMPOUNDRULEs.
func (h *Hunspell) ruleCompound(word string) bool {
	for _, r := range h.rules {
		if h.matchRule(r, word, 0, 0) {
			return true
		}
	}
	return false
}

// matchRule reports whether word[pos:] matches rule[i:].
func (h *Hunspell) matchRule(rule []ruleItem, word string, i, pos int) bool {
	if i == len(rule) {
		return pos == len(word)
	}
	it := rule[i]
	if it.q != 0 && h.matchRule(rule, word, i+1, pos) {
		return true
	}
	next := i + 1
	if it.q == '*' {
		next = i
	}
	for end := pos + 1; end <= len(word); end++ {
		if end < len(word) && !utf8.RuneStart(word[end]) {
			continue
		}
		if h.stemHas(word[pos:end], it.flag) && h.matchRule(rule, word, next, end) {
			return true
		}
	}
	return false
}

func (h *Hunspell) stemHas(word string, f flag) bool {
	for _, fs := range h.words[word] {
		if fs.has(f) {
			return true
		}
	}
	return false
}
