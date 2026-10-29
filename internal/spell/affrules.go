package spell

import (
	"strconv"
	"strings"
)

// affix reads a PFX or SFX line: the header of a class ("SFX A Y 3") or
// one of its entries ("SFX A y ies [^aeiou]y").
func (p *affParser) affix(f []string) {
	key := f[0] + " " + f[1]
	if p.left[key] == 0 {
		p.header(key, f)
		return
	}
	p.left[key]--
	if len(f) < 4 {
		return
	}
	a := p.entry(f)
	if f[0] == "PFX" {
		p.h.prefixes[a.add] = append(p.h.prefixes[a.add], a)
		p.h.maxPrefix = max(p.h.maxPrefix, len(a.add))
	} else {
		p.h.suffixes[a.add] = append(p.h.suffixes[a.add], a)
		p.h.maxSuffix = max(p.h.maxSuffix, len(a.add))
	}
}

func (p *affParser) header(key string, f []string) {
	if len(f) < 4 {
		return
	}
	n, err := strconv.Atoi(f[3])
	if err != nil {
		return
	}
	p.left[key] = n
	p.crossOf[key] = f[2] == "Y"
}

// entry parses the fields of an affix entry: flag, strip, add[/flags]
// and condition.
func (p *affParser) entry(f []string) *affix {
	a := &affix{flag: p.h.mode.one(f[1]), cross: p.crossOf[f[0]+" "+f[1]], strip: zero(f[2])}
	add, cont, _ := strings.Cut(f[3], "/")
	a.add = zero(add)
	if cont != "" {
		a.cont = p.h.flags(cont)
	}
	if len(f) > 4 {
		a.cond = parseCondition(f[4])
	}
	return a
}

// zero turns the "0" that stands for nothing into "".
func zero(s string) string {
	if s == "0" {
		return ""
	}
	return s
}

// compoundRule reads a COMPOUNDRULE: flags, in parentheses for long and
// numeric flags, each optionally followed by * or ?.
func (p *affParser) compoundRule(f []string) {
	if p.ruleCount == 0 {
		if n, err := strconv.Atoi(f[1]); err == nil {
			p.ruleCount = n
			return
		}
	}
	var rule []ruleItem
	for s := f[1]; s != ""; {
		var it ruleItem
		it, s = p.nextRuleItem(s)
		if it.q != 0 && len(rule) > 0 {
			rule[len(rule)-1].q = it.q
		} else if it.flag != 0 {
			rule = append(rule, it)
		}
	}
	if len(rule) > 0 {
		p.h.rules = append(p.h.rules, rule)
	}
}

// nextRuleItem returns a flag, or a quantifier in q, and the rest of s.
func (p *affParser) nextRuleItem(s string) (ruleItem, string) {
	switch {
	case s[0] == '*' || s[0] == '?':
		return ruleItem{q: rune(s[0])}, s[1:]
	case s[0] == '(':
		end := strings.IndexByte(s, ')')
		if end < 0 {
			return ruleItem{}, ""
		}
		return ruleItem{flag: p.h.mode.one(s[1:end])}, s[end+1:]
	}
	r := []rune(s)[0]
	return ruleItem{flag: flag(r)}, s[len(string(r)):]
}
