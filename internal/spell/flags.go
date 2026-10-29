package spell

import (
	"slices"
	"strconv"
	"strings"
)

// flag is a Hunspell flag: a name given to an affix class or a property
// (such as "may start a compound") that dictionary words are marked with.
// 0 means "no flag".
type flag uint32

// flagSet is a sorted set of flags.
type flagSet []flag

func (s flagSet) has(f flag) bool {
	if f == 0 {
		return false
	}
	_, ok := slices.BinarySearch(s, f)
	return ok
}

// flagMode is how flags are written in a dictionary (the FLAG directive).
type flagMode int

const (
	flagChar flagMode = iota // one character each (the default, and UTF-8)
	flagLong                 // two characters each
	flagNum                  // decimal numbers separated by commas
)

// parse splits a run of flags.
func (m flagMode) parse(s string) flagSet {
	var out flagSet
	switch m {
	case flagLong:
		out = parseLongFlags(s)
	case flagNum:
		out = parseNumFlags(s)
	default:
		for _, r := range s {
			out = append(out, flag(r))
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func parseLongFlags(s string) flagSet {
	var out flagSet
	rs := []rune(s)
	for i := 0; i+1 < len(rs); i += 2 {
		out = append(out, flag(rs[i])<<16|flag(rs[i+1]))
	}
	return out
}

func parseNumFlags(s string) flagSet {
	var out flagSet
	for _, part := range strings.Split(s, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && n > 0 {
			out = append(out, flag(n))
		}
	}
	return out
}

// one parses a single flag, as given to directives such as NEEDAFFIX.
func (m flagMode) one(s string) flag {
	if fs := m.parse(s); len(fs) > 0 {
		return fs[0]
	}
	return 0
}
