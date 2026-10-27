// Package tex typesets a practical subset of TeX/LaTeX maths, following
// TeX's layout rules (simplified), into glyphs, rules and polygons that the
// preview and the PDF exporter both draw.
package tex

import (
	"strconv"
	"strings"
	"unicode"
)

type node interface{}

type (
	nSym struct {
		r    rune
		cls  class
		font fontStyle
	}
	nGroup   struct{ items []node }
	nScripts struct {
		base, sup, sub node
		limits         int8 // 1 forces limits, -1 forbids them, 0 automatic
	}
	nOp struct {
		r      rune   // big operator, or
		name   string // operator name set upright
		limits bool
	}
	nFrac struct {
		num, den    node
		style       int8 // -1 automatic, 0 display, 1 text
		bar         bool
		left, right rune
	}
	nSqrt      struct{ index, body node }
	nLeftRight struct {
		delims []rune // left, middles..., right
		parts  [][]node
	}
	nAccent struct {
		kind string
		body node
	}
	nSpace struct{ w float32 }
	nText  struct {
		s    string
		font fontStyle
	}
	nArray struct {
		env  string
		cols string
		rows [][][]node
	}
	nBig struct {
		r    rune
		size float32
		cls  class
	}
	nStyle struct{ st style }
	nError struct{ s string }
)

type parser struct {
	s    []rune
	i    int
	font fontStyle
}

// parse returns the rows and cells of the expression; a plain expression is
// a single row with a single cell.
func parse(src string) [][][]node {
	p := &parser{s: []rune(src)}
	return p.rows("")
}

func (p *parser) eof() bool { return p.i >= len(p.s) }

func (p *parser) skipSpace() {
	for !p.eof() {
		c := p.s[p.i]
		if c == '%' {
			for !p.eof() && p.s[p.i] != '\n' {
				p.i++
			}
			continue
		}
		if !unicode.IsSpace(c) {
			return
		}
		p.i++
	}
}

// peekCmd returns the name of the control sequence at the current position,
// or "" if there is none.
func (p *parser) peekCmd() string {
	if p.eof() || p.s[p.i] != '\\' || p.i+1 >= len(p.s) {
		return ""
	}
	j := p.i + 1
	if !isLetter(p.s[j]) {
		return string(p.s[j])
	}
	for j < len(p.s) && isLetter(p.s[j]) {
		j++
	}
	return string(p.s[p.i+1 : j])
}

func isLetter(r rune) bool { return r < 128 && unicode.IsLetter(r) }

func (p *parser) readCmd() string {
	name := p.peekCmd()
	p.i += 1 + len([]rune(name))
	if name == "" && !p.eof() {
		p.i++ // lone backslash at the end
	}
	return name
}

func isTerminator(name string) bool {
	switch name {
	case "\\", "cr", "right", "middle", "end":
		return true
	}
	return false
}

// list parses atoms until the end of input, '}', '&' or a terminating
// control sequence, which is left unconsumed.
func (p *parser) list() []node {
	var out []node
	for {
		p.skipSpace()
		if p.eof() {
			return out
		}
		c := p.s[p.i]
		if c == '}' || c == '&' {
			return out
		}
		if c == '\\' && isTerminator(p.peekCmd()) {
			return out
		}
		switch c {
		case '^', '_':
			p.i++
			arg := p.arg()
			sc := p.scriptsFor(&out, c == '^')
			if c == '^' {
				sc.sup = arg
			} else {
				sc.sub = arg
			}
			continue
		case '\'':
			n := 0
			for !p.eof() && p.s[p.i] == '\'' {
				n++
				p.i++
			}
			primes := []rune{'′', '″', '‴', '⁗'}[min(n, 4)-1]
			sc := p.scriptsFor(&out, true)
			prime := &nSym{r: primes, cls: clsOrd, font: fRoman}
			if sc.sup == nil {
				sc.sup = prime
			} else {
				sc.sup = &nGroup{items: []node{prime, sc.sup}}
			}
			continue
		}
		if c == '\\' {
			switch name := p.peekCmd(); name {
			case "limits", "nolimits":
				p.readCmd()
				if len(out) > 0 {
					sc, ok := out[len(out)-1].(*nScripts)
					if !ok {
						sc = &nScripts{base: out[len(out)-1]}
						out[len(out)-1] = sc
					}
					sc.limits = map[string]int8{"limits": 1, "nolimits": -1}[name]
				}
				continue
			}
		}
		if n := p.atom(); n != nil {
			out = append(out, n)
		}
	}
}

// scriptsFor returns the scripts node to attach a superscript (sup) or
// subscript to, wrapping the last atom if needed.
func (p *parser) scriptsFor(out *[]node, sup bool) *nScripts {
	if n := len(*out); n > 0 {
		if sc, ok := (*out)[n-1].(*nScripts); ok && ((sup && sc.sup == nil) || (!sup && sc.sub == nil)) {
			return sc
		}
		sc := &nScripts{base: (*out)[n-1]}
		(*out)[n-1] = sc
		return sc
	}
	sc := &nScripts{base: &nGroup{}}
	*out = append(*out, sc)
	return sc
}

// arg parses a macro argument: a braced group or a single atom.
func (p *parser) arg() node {
	p.skipSpace()
	if p.eof() {
		return &nGroup{}
	}
	if p.s[p.i] == '{' {
		return p.group()
	}
	if p.s[p.i] == '}' {
		return &nGroup{}
	}
	return p.atom()
}

func (p *parser) group() node {
	p.i++ // {
	font := p.font
	items := p.list()
	p.font = font
	if !p.eof() && p.s[p.i] == '}' {
		p.i++
	}
	return &nGroup{items: items}
}

// rawGroup returns the text of a braced argument without parsing it.
func (p *parser) rawGroup() string {
	p.skipSpace()
	if p.eof() {
		return ""
	}
	if p.s[p.i] != '{' {
		r := p.s[p.i]
		p.i++
		return string(r)
	}
	depth := 0
	start := p.i + 1
	for ; !p.eof(); p.i++ {
		switch p.s[p.i] {
		case '\\':
			p.i++
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				s := string(p.s[start:p.i])
				p.i++
				return s
			}
		}
	}
	return string(p.s[start:])
}

// optional returns the text of an optional [argument], if present.
func (p *parser) optional() (string, bool) {
	p.skipSpace()
	if p.eof() || p.s[p.i] != '[' {
		return "", false
	}
	start := p.i + 1
	depth := 0
	for p.i++; !p.eof(); p.i++ {
		switch p.s[p.i] {
		case '{':
			depth++
		case '}':
			depth--
		case ']':
			if depth == 0 {
				s := string(p.s[start:p.i])
				p.i++
				return s, true
			}
		}
	}
	return string(p.s[start:]), true
}

func (p *parser) sub(src string) node {
	q := &parser{s: []rune(src), font: p.font}
	return &nGroup{items: q.list()}
}

var charClasses = map[rune]class{
	'+': clsBin, '−': clsBin, '∗': clsBin, '=': clsRel, '<': clsRel, '>': clsRel, ':': clsRel,
	',': clsPunct, ';': clsPunct, '(': clsOpen, '[': clsOpen, ')': clsClose, ']': clsClose,
	'!': clsClose, '?': clsClose, '|': clsOrd, '‖': clsOrd, '/': clsOrd, '.': clsOrd,
}

func (p *parser) atom() node {
	c := p.s[p.i]
	switch {
	case c == '{':
		return p.group()
	case c == '\\':
		return p.command()
	case c == '~':
		p.i++
		return &nSpace{w: 0.33}
	}
	p.i++
	switch {
	case isLetter(c) || (c >= 'α' && c <= 'ω'):
		return &nSym{r: c, cls: clsOrd, font: p.font}
	case c >= '0' && c <= '9':
		f := p.font
		if f == fItalic {
			f = fRoman
		}
		return &nSym{r: c, cls: clsOrd, font: f}
	}
	switch c {
	case '-':
		c = '−'
	case '*':
		c = '∗'
	}
	cls, ok := charClasses[c]
	if !ok {
		cls = classOfRune(c)
	}
	return &nSym{r: c, cls: cls, font: fRoman}
}

var runeClasses map[rune]class

// classOfRune gives typed Unicode symbols (≤, →, ∑ …) their usual class.
func classOfRune(r rune) class {
	if runeClasses == nil {
		runeClasses = map[rune]class{}
		for _, s := range symbols {
			if c, seen := runeClasses[s.r]; seen && c != s.cls {
				// Shared by \lvert, \rvert …: use the neutral class.
				runeClasses[s.r] = clsOrd
				continue
			}
			runeClasses[s.r] = s.cls
		}
	}
	if c, ok := runeClasses[r]; ok {
		return c
	}
	return clsOrd
}

var negations = map[rune]rune{
	'=': '≠', '∈': '∉', '<': '≮', '>': '≯', '≤': '≰', '≥': '≱', '⊂': '⊄', '⊃': '⊅',
	'⊆': '⊈', '⊇': '⊉', '∼': '≁', '≡': '≢', '≈': '≉', '∃': '∄', '∣': '∤', '∥': '∦', '≅': '≇',
}

func (p *parser) command() node {
	name := p.readCmd()
	if w, ok := spaces[name]; ok {
		return &nSpace{w: w}
	}
	if s, ok := symbols[name]; ok {
		f := fRoman
		if s.r >= 'α' && s.r <= 'ω' || italicGreekVariants[s.r] != 0 {
			f = p.font
		}
		return &nSym{r: s.r, cls: s.cls, font: f}
	}
	if op, ok := bigOps[name]; ok {
		return &nOp{r: op.r, limits: op.limits}
	}
	if limits, ok := functions[name]; ok {
		n := name
		if dn, ok := functionNames[name]; ok {
			n = dn
		}
		return &nOp{name: n, limits: limits}
	}
	if f, ok := fontCommands[name]; ok {
		old := p.font
		p.font = f
		body := p.arg()
		p.font = old
		return body
	}
	if r, ok := accents[name]; ok {
		_ = r
		return &nAccent{kind: name, body: p.arg()}
	}
	switch name {
	case "frac", "dfrac", "tfrac", "cfrac":
		style := map[string]int8{"frac": -1, "dfrac": 0, "cfrac": 0, "tfrac": 1}[name]
		return &nFrac{num: p.arg(), den: p.arg(), bar: true, style: style}
	case "binom", "dbinom", "tbinom":
		style := map[string]int8{"binom": -1, "dbinom": 0, "tbinom": 1}[name]
		return &nFrac{num: p.arg(), den: p.arg(), style: style, left: '(', right: ')'}
	case "sqrt":
		var index node
		if s, ok := p.optional(); ok {
			index = p.sub(s)
		}
		return &nSqrt{index: index, body: p.arg()}
	case "left":
		lr := &nLeftRight{delims: []rune{p.delim()}}
		for {
			lr.parts = append(lr.parts, p.list())
			switch p.peekCmd() {
			case "middle":
				p.readCmd()
				lr.delims = append(lr.delims, p.delim())
				continue
			case "right":
				p.readCmd()
				lr.delims = append(lr.delims, p.delim())
			default:
				// Unclosed \left, or a stray terminator: close it here.
				if !p.eof() && p.s[p.i] == '}' {
					p.i++
				}
				lr.delims = append(lr.delims, 0)
			}
			return lr
		}
	case "big", "Big", "bigg", "Bigg", "bigl", "Bigl", "biggl", "Biggl",
		"bigr", "Bigr", "biggr", "Biggr", "bigm", "Bigm", "biggm", "Biggm":
		size := map[string]float32{"big": 1.2, "Big": 1.8, "bigg": 2.4, "Bigg": 3.0}[strings.TrimRight(name, "lrm")]
		cls := clsOrd
		switch name[len(name)-1] {
		case 'l':
			cls = clsOpen
		case 'r':
			cls = clsClose
		case 'm':
			cls = clsRel
		}
		return &nBig{r: p.delim(), size: size, cls: cls}
	case "bar", "overline", "underline", "vec", "overrightarrow", "overleftarrow",
		"overbrace", "underbrace", "boxed", "cancel", "phantom", "hphantom", "vphantom":
		return &nAccent{kind: name, body: p.arg()}
	case "not":
		p.skipSpace()
		if p.eof() {
			return &nSym{r: '/', cls: clsRel, font: fRoman}
		}
		n := p.atom()
		if s, ok := n.(*nSym); ok {
			if neg, ok := negations[s.r]; ok {
				return &nSym{r: neg, cls: clsRel, font: fRoman}
			}
		}
		return &nAccent{kind: "not", body: n}
	case "text", "textrm", "textnormal", "mbox", "hbox", "textup", "textsf", "texttt":
		return &nText{s: p.rawGroup(), font: fRoman}
	case "textit", "emph":
		return &nText{s: p.rawGroup(), font: fItalic}
	case "textbf":
		return &nText{s: p.rawGroup(), font: fBold}
	case "operatorname", "operatornamewithlimits":
		limits := name == "operatornamewithlimits"
		if !p.eof() && p.s[p.i] == '*' {
			p.i++
			limits = true
		}
		return &nOp{name: p.rawGroup(), limits: limits}
	case "displaystyle":
		return &nStyle{st: sD}
	case "textstyle":
		return &nStyle{st: sT}
	case "scriptstyle":
		return &nStyle{st: sS}
	case "scriptscriptstyle":
		return &nStyle{st: sSS}
	case "overset", "stackrel":
		over := p.arg()
		return &nScripts{base: p.arg(), sup: over, limits: 1}
	case "underset":
		under := p.arg()
		return &nScripts{base: p.arg(), sub: under, limits: 1}
	case "begin":
		env := strings.TrimSpace(p.rawGroup())
		a := &nArray{env: strings.TrimSuffix(env, "*")}
		if a.env == "array" {
			a.cols = p.rawGroup()
		}
		a.rows = p.rows(env)
		return a
	case "pmod":
		arg := p.arg()
		return &nGroup{items: []node{&nSpace{w: 1}, &nSym{r: '(', cls: clsOpen, font: fRoman},
			&nOp{name: "mod"}, &nSpace{w: 6.0 / 18}, arg, &nSym{r: ')', cls: clsClose, font: fRoman}}}
	case "bmod":
		return &nGroup{items: []node{&nSpace{w: 4.0 / 18}, &nOp{name: "mod"}, &nSpace{w: 4.0 / 18}}}
	case "mod":
		return &nGroup{items: []node{&nSpace{w: 1}, &nOp{name: "mod"}, &nSpace{w: 6.0 / 18}}}
	case "color":
		p.rawGroup()
		return nil
	case "textcolor":
		p.rawGroup()
		return p.arg()
	case "tag", "tag*":
		return &nGroup{items: []node{&nSpace{w: 2}, &nText{s: "(" + p.rawGroup() + ")", font: fRoman}}}
	case "hspace", "kern", "mkern", "mskip", "hskip":
		return &nSpace{w: parseLength(p.rawGroup())}
	case "label", "nonumber", "notag":
		if name == "label" {
			p.rawGroup()
		}
		return nil
	case "right", "middle", "end":
		return &nError{s: "\\" + name}
	}
	if name == "" {
		return nil
	}
	return &nError{s: "\\" + name}
}

// parseLength converts a TeX length like "1em", "3mu" or "5pt" to em.
func parseLength(s string) float32 {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) && (s[i] == '-' || s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	v, err := strconv.ParseFloat(s[:i], 32)
	if err != nil {
		return 0
	}
	switch strings.TrimSpace(s[i:]) {
	case "mu":
		v /= 18
	case "pt":
		v /= 10
	case "ex":
		v *= 0.43
	case "cm":
		v *= 2.85
	case "mm":
		v *= 0.285
	}
	return float32(v)
}

// delim reads a delimiter after \left, \right, \big… (0 is the null
// delimiter ".").
func (p *parser) delim() rune {
	p.skipSpace()
	if p.eof() {
		return 0
	}
	c := p.s[p.i]
	if c == '\\' {
		name := p.readCmd()
		if r, ok := delimiterNames[name]; ok {
			return r
		}
		if s, ok := symbols[name]; ok {
			return s.r
		}
		return 0
	}
	p.i++
	switch c {
	case '.':
		return 0
	case '<':
		return '⟨'
	case '>':
		return '⟩'
	}
	return c
}

// rows parses the body of an environment (or the whole input when env is
// ""), split into rows by \\ and cells by &.
func (p *parser) rows(env string) [][][]node {
	var rows [][][]node
	var row [][]node
	var cell []node
loop:
	for {
		cell = append(cell, p.list()...)
		if p.eof() {
			break
		}
		if p.s[p.i] == '&' {
			p.i++
			row = append(row, cell)
			cell = nil
			continue
		}
		if p.s[p.i] == '}' {
			p.i++ // stray closing brace
			continue
		}
		switch name := p.readCmd(); name {
		case "\\", "cr":
			p.optional() // \\[2pt]
			rows = append(rows, append(row, cell))
			row, cell = nil, nil
		case "end":
			p.rawGroup()
			if env != "" {
				break loop
			}
		default: // stray \right or \middle
			cell = append(cell, &nError{s: "\\" + name})
		}
	}
	rows = append(rows, append(row, cell))
	// Drop a trailing empty row left by a final \\.
	if n := len(rows); n > 1 && len(rows[n-1]) == 1 && len(rows[n-1][0]) == 0 {
		rows = rows[:n-1]
	}
	return rows
}
