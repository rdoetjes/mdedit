package spell

// form is one way to read a word: a dictionary stem's flags plus the
// affixes taken off to reach it. sfx is the outer suffix, sfx2 an inner
// one (Hunspell's "two-fold suffixes").
type form struct {
	flags          flagSet
	pfx, sfx, sfx2 *affix
}

// has reports whether the stem or one of the affixes carries f.
func (f form) has(fl flag) bool {
	return f.flags.has(fl) || f.contHas(fl)
}

// contHas reports whether one of the affixes carries f.
func (f form) contHas(fl flag) bool {
	for _, a := range []*affix{f.pfx, f.sfx, f.sfx2} {
		if a != nil && a.cont.has(fl) {
			return true
		}
	}
	return false
}

func (f form) affixes() int {
	n := 0
	for _, a := range []*affix{f.pfx, f.sfx, f.sfx2} {
		if a != nil {
			n++
		}
	}
	return n
}

// forms returns every reading of word as a stem with affixes.
func (h *Hunspell) forms(word string) []form {
	var out []form
	for _, fs := range h.words[word] {
		out = append(out, form{flags: fs})
	}
	out = h.suffixForms(out, word, nil)
	return h.prefixForms(out, word)
}

// suffixForms appends the readings of word as stem + suffix (+ suffix).
// With pfx set, word has had that prefix removed and the suffix must
// combine with it.
func (h *Hunspell) suffixForms(out []form, word string, pfx *affix) []form {
	for n := 0; n <= min(h.maxSuffix, len(word)); n++ {
		for _, s := range h.suffixes[word[len(word)-n:]] {
			stem, ok := stripSuffix(word, s)
			if !ok || (pfx != nil && !s.cross) {
				continue
			}
			out = h.stemsWith(out, stem, form{pfx: pfx, sfx: s})
			out = h.innerSuffixForms(out, stem, form{pfx: pfx, sfx: s})
		}
	}
	return out
}

// stripSuffix undoes suffix s on word, if s could have made it.
func stripSuffix(word string, s *affix) (string, bool) {
	stem := word[:len(word)-len(s.add)] + s.strip
	return stem, stem != "" && s.cond.matchEnd(stem)
}

// innerSuffixForms finds a second suffix under the outer one in f: stem
// is then itself a stem plus a suffix whose continuation allows f.sfx.
func (h *Hunspell) innerSuffixForms(out []form, word string, f form) []form {
	for n := 0; n <= min(h.maxSuffix, len(word)); n++ {
		for _, s := range h.suffixes[word[len(word)-n:]] {
			if !s.cont.has(f.sfx.flag) {
				continue
			}
			if stem, ok := stripSuffix(word, s); ok {
				out = h.stemsWith(out, stem, form{pfx: f.pfx, sfx: f.sfx, sfx2: s})
			}
		}
	}
	return out
}

// stemsWith appends f for each entry of stem allowing all of f's affixes.
// The outer suffix of a two-fold pair is allowed by the inner one, not by
// the stem.
func (h *Hunspell) stemsWith(out []form, stem string, f form) []form {
	for _, fs := range h.words[stem] {
		if f.pfx != nil && !fs.has(f.pfx.flag) {
			continue
		}
		if (f.sfx2 != nil && fs.has(f.sfx2.flag)) || (f.sfx2 == nil && fs.has(f.sfx.flag)) {
			f.flags = fs
			out = append(out, f)
		}
	}
	return out
}

// prefixForms appends the readings of word as prefix + stem, with or
// without a suffix.
func (h *Hunspell) prefixForms(out []form, word string) []form {
	for n := 1; n <= min(h.maxPrefix, len(word)); n++ {
		for _, p := range h.prefixes[word[:n]] {
			out = h.prefixedForms(out, p, p.strip+word[n:])
		}
	}
	return out
}

// prefixedForms appends the readings of rest, a word with prefix p taken
// off, as a stem allowing p, with or without a suffix.
func (h *Hunspell) prefixedForms(out []form, p *affix, rest string) []form {
	if rest == "" || !p.cond.matchStart(rest) {
		return out
	}
	for _, fs := range h.words[rest] {
		if fs.has(p.flag) {
			out = append(out, form{flags: fs, pfx: p})
		}
	}
	if p.cross {
		out = h.suffixForms(out, rest, p)
	}
	return out
}

// validAlone reports whether a reading is a word on its own, outside a
// compound.
func (h *Hunspell) validAlone(f form) bool {
	s := h.special
	if f.has(s.onlyInCompound) {
		return false
	}
	if f.flags.has(s.forbidden) && f.affixes() == 0 {
		return false
	}
	return h.affixesComplete(f)
}

// affixesComplete checks NEEDAFFIX (a stem or affix that needs another
// affix) and CIRCUMFIX (affixes that only come as prefix-suffix pairs).
func (h *Hunspell) affixesComplete(f form) bool {
	s := h.special
	n := f.affixes()
	if f.flags.has(s.needAffix) && n == 0 {
		return false
	}
	for _, a := range []*affix{f.pfx, f.sfx, f.sfx2} {
		if a != nil && a.cont.has(s.needAffix) && n < 2 {
			return false
		}
	}
	if s.circumfix == 0 {
		return true
	}
	pc := f.pfx != nil && f.pfx.cont.has(s.circumfix)
	sc := (f.sfx != nil && f.sfx.cont.has(s.circumfix)) || (f.sfx2 != nil && f.sfx2.cont.has(s.circumfix))
	return pc == sc
}
