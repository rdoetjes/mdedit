package spell

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// affParser reads an .aff file into a Hunspell.
type affParser struct {
	h *Hunspell
	// left counts the entries still to come for each affix class, so a
	// line can be told apart from the header of the next class.
	left map[string]int
	// crossOf is the cross-product setting of each affix class.
	crossOf map[string]bool
	// ruleCount is the number of COMPOUNDRULE lines announced.
	ruleCount int
	err       error
}

// affDirectives maps the directives mdedit understands to their handlers.
var affDirectives = map[string]func(*affParser, []string){
	"SET":          (*affParser).set,
	"FLAG":         (*affParser).flagMode,
	"AF":           (*affParser).alias,
	"PFX":          (*affParser).affix,
	"SFX":          (*affParser).affix,
	"COMPOUNDMIN":  (*affParser).compoundMin,
	"COMPOUNDRULE": (*affParser).compoundRule,
	"ICONV":        (*affParser).iconv,
	"REP":          (*affParser).rep,
	"TRY":          (*affParser).tryChars,
}

// errCharset reports a dictionary in an encoding other than UTF-8.
type errCharset string

func (e errCharset) Error() string {
	return fmt.Sprintf("dictionary encoding %s is not supported (only UTF-8)", string(e))
}

func (p *affParser) parse(r io.Reader) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for p.err == nil && sc.Scan() {
		p.line(sc.Text())
	}
	if p.err != nil {
		return p.err
	}
	return sc.Err()
}

func (p *affParser) line(line string) {
	line = strings.TrimPrefix(line, "\uFEFF")
	fields := strings.Fields(line)
	if len(fields) < 2 || strings.HasPrefix(fields[0], "#") {
		return
	}
	if handle, ok := affDirectives[fields[0]]; ok {
		handle(p, fields)
		return
	}
	if ptr := p.h.special.ptr(fields[0]); ptr != nil {
		*ptr = p.h.mode.one(fields[1])
	}
}

func (p *affParser) set(f []string) {
	if enc := strings.ToUpper(f[1]); enc != "UTF-8" && enc != "UTF8" {
		p.err = errCharset(f[1])
	}
}

func (p *affParser) flagMode(f []string) {
	switch strings.ToLower(f[1]) {
	case "long":
		p.h.mode = flagLong
	case "num":
		p.h.mode = flagNum
	default:
		p.h.mode = flagChar
	}
}

// alias reads AF lines: numbered flag sets that the .dic file and affix
// rules may refer to by number.
func (p *affParser) alias(f []string) {
	if _, err := strconv.Atoi(f[1]); err == nil && p.h.aliases == nil {
		p.h.aliases = []flagSet{} // the count line
		return
	}
	p.h.aliases = append(p.h.aliases, p.h.mode.parse(f[1]))
}

// flags parses a flag field, which is an AF number if AF is used.
func (h *Hunspell) flags(s string) flagSet {
	if len(h.aliases) > 0 {
		if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= len(h.aliases) {
			return h.aliases[n-1]
		}
	}
	return h.mode.parse(s)
}

func (p *affParser) compoundMin(f []string) {
	if n, err := strconv.Atoi(f[1]); err == nil {
		p.h.compoundMin = max(n, 1)
	}
}

func (p *affParser) iconv(f []string) {
	if len(f) >= 3 {
		p.h.iconv = append(p.h.iconv, [2]string{f[1], f[2]})
	}
}

func (p *affParser) tryChars(f []string) {
	p.h.try = []rune(f[1])
}

// rep reads a REP entry; "_" stands for a space, ^ and $ anchor it.
func (p *affParser) rep(f []string) {
	if len(f) < 3 {
		return // the count line
	}
	from, to := strings.ReplaceAll(f[1], "_", " "), strings.ReplaceAll(f[2], "_", " ")
	r := repPattern{to: to}
	from, r.start = strings.CutPrefix(from, "^")
	r.from, r.end = strings.CutSuffix(from, "$")
	if r.from != "" {
		p.h.rep = append(p.h.rep, r)
	}
}
