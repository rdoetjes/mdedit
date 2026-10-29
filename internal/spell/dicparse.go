package spell

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// LoadHunspell reads a dictionary from its .aff and .dic files.
func LoadHunspell(aff, dic io.Reader) (*Hunspell, error) {
	h := &Hunspell{
		words:       make(map[string][]flagSet),
		prefixes:    make(map[string][]*affix),
		suffixes:    make(map[string][]*affix),
		compoundMin: 3,
	}
	p := &affParser{h: h, left: make(map[string]int), crossOf: make(map[string]bool)}
	if err := p.parse(aff); err != nil {
		return nil, err
	}
	if err := h.readDic(dic); err != nil {
		return nil, err
	}
	h.stems = h.standaloneStems()
	return h, nil
}

// readDic reads the word list: a count, then one "word/flags" per line,
// possibly followed by morphological fields.
func (h *Hunspell) readDic(r io.Reader) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	interned := make(map[string]flagSet)
	first := true
	for sc.Scan() {
		line := strings.TrimPrefix(sc.Text(), "\uFEFF")
		if first {
			first = false
			if _, err := strconv.Atoi(strings.TrimSpace(line)); err == nil {
				continue
			}
		}
		h.dicLine(line, interned)
	}
	return sc.Err()
}

// dicLine adds one word. Flag sets are shared between words with the same
// flags, which saves a lot of memory.
func (h *Hunspell) dicLine(line string, interned map[string]flagSet) {
	fields := strings.Fields(line)
	if len(fields) == 0 || strings.HasPrefix(line, "#") {
		return
	}
	word, flags := splitDicWord(fields[0])
	fs, ok := interned[flags]
	if !ok {
		fs = h.flags(flags)
		interned[flags] = fs
	}
	h.words[word] = append(h.words[word], fs)
}

// splitDicWord splits "word/flags" at the first slash that isn't escaped
// as "\/".
func splitDicWord(s string) (string, string) {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '/':
			return strings.ReplaceAll(s[:i], `\/`, "/"), s[i+1:]
		}
	}
	return strings.ReplaceAll(s, `\/`, "/"), ""
}

// standaloneStems collects the stems that are words on their own, as
// candidates for suggestions.
func (h *Hunspell) standaloneStems() *Dictionary {
	d := New()
	for w, sets := range h.words {
		for _, fs := range sets {
			if h.validAlone(form{flags: fs}) && !fs.has(h.special.forbidden) {
				d.add(w, 0)
				break
			}
		}
	}
	return d
}
