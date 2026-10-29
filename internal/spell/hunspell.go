package spell

import (
	"strings"
	"sync"
)

// Hunspell is a dictionary in the Hunspell format used by LibreOffice,
// Firefox and most Linux programs: a list of word stems (the .dic file),
// each marked with flags, and rules (the .aff file) saying which prefixes
// and suffixes each flag allows and which words may be joined into compound
// words.
//
// Words are checked by stripping affixes until a stem is found, not by
// expanding every form in advance: that would take hundreds of megabytes
// for languages like Dutch or Polish, and compound words can't be expanded
// at all.
//
// Supported: FLAG (characters, long, num), AF, PFX and SFX with conditions,
// cross products and two suffixes, NEEDAFFIX, FORBIDDENWORD,
// ONLYINCOMPOUND, CIRCUMFIX, COMPOUNDFLAG, COMPOUNDBEGIN, COMPOUNDMIDDLE,
// COMPOUNDEND, COMPOUNDPERMITFLAG, COMPOUNDMIN, COMPOUNDRULE, ICONV, REP
// and TRY. Other directives, mostly finer compound checks, are ignored, so
// a few wrong compounds may be accepted.
type Hunspell struct {
	mode    flagMode
	aliases []flagSet // AF
	words   map[string][]flagSet
	// Affixes by the text they add, and the longest such text in bytes.
	prefixes, suffixes   map[string][]*affix
	maxPrefix, maxSuffix int
	special              specials
	compoundMin          int
	rules                [][]ruleItem
	iconv                [][2]string
	rep                  []repPattern
	try                  []rune
	// stems are the words that are correct on their own, for suggestions.
	stems *Dictionary

	mu    sync.Mutex
	cache map[string]bool
}

// affix is one prefix or suffix rule.
type affix struct {
	flag       flag
	cross      bool // may combine with an affix of the other kind
	strip, add string
	cond       condition
	cont       flagSet // flags the affixed word gets (continuation classes)
}

// specials are the flags with a meaning of their own.
type specials struct {
	forbidden, needAffix, onlyInCompound, circumfix flag
	compound, begin, middle, end, permit            flag
}

// ptr returns the special flag set by a directive, or nil.
func (s *specials) ptr(directive string) *flag {
	switch directive {
	case "FORBIDDENWORD":
		return &s.forbidden
	case "NEEDAFFIX", "PSEUDOROOT":
		return &s.needAffix
	case "ONLYINCOMPOUND":
		return &s.onlyInCompound
	case "CIRCUMFIX":
		return &s.circumfix
	case "COMPOUNDFLAG":
		return &s.compound
	case "COMPOUNDBEGIN":
		return &s.begin
	case "COMPOUNDMIDDLE":
		return &s.middle
	case "COMPOUNDEND":
		return &s.end
	case "COMPOUNDPERMITFLAG":
		return &s.permit
	}
	return nil
}

// compounds reports whether the dictionary forms compound words.
func (h *Hunspell) compounds() bool {
	s := h.special
	return s.compound != 0 || s.begin != 0 || s.end != 0
}

// repPattern is a REP entry: a common mistake and its correction, used for
// suggestions. Anchored patterns only match at the start or end.
type repPattern struct {
	from, to   string
	start, end bool
}

// ruleItem is one element of a COMPOUNDRULE: a flag, optionally repeated
// ('*' for any number, '?' for at most one).
type ruleItem struct {
	flag flag
	q    rune
}

// Correct reports whether word is spelt correctly: a dictionary word, an
// affixed form of one, or a compound of them. A capitalised or upper-case
// word is also correct when its lower-case form is.
func (h *Hunspell) Correct(word string) bool {
	word = h.convert(normalise(word))
	h.mu.Lock()
	defer h.mu.Unlock()
	if ok, cached := h.cache[word]; cached {
		return ok
	}
	ok := h.checkCased(word)
	if len(h.cache) > 200000 {
		h.cache = nil
	}
	if h.cache == nil {
		h.cache = make(map[string]bool)
	}
	h.cache[word] = ok
	return ok
}

func (h *Hunspell) checkCased(word string) bool {
	if h.check(word) {
		return true
	}
	lower := strings.ToLower(word)
	if lower == word {
		return false
	}
	if h.check(lower) {
		return true
	}
	return isUpper(word) && h.check(capitalise(lower))
}

// convert applies the ICONV table, which maps typographic variants (curly
// apostrophes, ligatures) to the dictionary's spelling.
func (h *Hunspell) convert(word string) string {
	for _, c := range h.iconv {
		word = strings.ReplaceAll(word, c[0], c[1])
	}
	return word
}

// check checks a word as written.
func (h *Hunspell) check(word string) bool {
	if h.forbiddenWord(word) {
		return false
	}
	for _, f := range h.forms(word) {
		if h.validAlone(f) {
			return true
		}
	}
	return h.compounds() && h.compoundFrom(word, 0, 0) || h.ruleCompound(word) || h.elision(word)
}

// elision accepts an elided word such as French "l'homme" or "qu'il":
// dictionaries list "l'" and "qu'" as words and rely on the word processor
// to split them off.
func (h *Hunspell) elision(word string) bool {
	i := strings.IndexByte(word, '\'')
	if i <= 0 || i == len(word)-1 {
		return false
	}
	return h.check(word[:i+1]) && h.checkCased(word[i+1:])
}

// forbiddenWord reports whether the dictionary marks word as wrong.
func (h *Hunspell) forbiddenWord(word string) bool {
	for _, fs := range h.words[word] {
		if fs.has(h.special.forbidden) {
			return true
		}
	}
	return false
}
