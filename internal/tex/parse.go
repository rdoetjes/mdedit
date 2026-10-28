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

// skipSpace skips white space and % comments.
func (p *parser) skipSpace() {
	for !p.eof() {
		switch c := p.s[p.i]; {
		case c == '%':
			p.skipLine()
		case unicode.IsSpace(c):
			p.i++
		default:
			return
		}
	}
}

func (p *parser) skipLine() {
	for !p.eof() && p.s[p.i] != '\n' {
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
	for p.skipSpace(); !p.atListEnd(); p.skipSpace() {
		p.listItem(&out)
	}
	return out
}

func (p *parser) atListEnd() bool {
	if p.eof() {
		return true
	}
	c := p.s[p.i]
	return c == '}' || c == '&' || (c == '\\' && isTerminator(p.peekCmd()))
}

// listItem parses one atom, or a modifier of the previous one.
func (p *parser) listItem(out *[]node) {
	c := p.s[p.i]
	switch {
	case c == '^' || c == '_':
		p.i++
		p.attachScript(out, c == '^', p.arg())
	case c == '\'':
		p.primes(out)
	case p.peekCmd() == "limits" || p.peekCmd() == "nolimits":
		p.limits(out, p.readCmd())
	default:
		if n := p.atom(); n != nil {
			*out = append(*out, n)
		}
	}
}

func (p *parser) attachScript(out *[]node, sup bool, arg node) {
	sc := p.scriptsFor(out, sup)
	if sup {
		sc.sup = arg
	} else {
		sc.sub = arg
	}
}

// primes turns x' and x” into superscript primes.
func (p *parser) primes(out *[]node) {
	n := 0
	for !p.eof() && p.s[p.i] == '\'' {
		n++
		p.i++
	}
	prime := &nSym{r: []rune{'′', '″', '‴', '⁗'}[min(n, 4)-1], cls: clsOrd, font: fRoman}
	sc := p.scriptsFor(out, true)
	if sc.sup == nil {
		sc.sup = prime
	} else {
		sc.sup = &nGroup{items: []node{prime, sc.sup}}
	}
}

// limits applies \limits or \nolimits to the previous atom.
func (p *parser) limits(out *[]node, name string) {
	n := len(*out)
	if n == 0 {
		return
	}
	sc, ok := (*out)[n-1].(*nScripts)
	if !ok {
		sc = &nScripts{base: (*out)[n-1]}
		(*out)[n-1] = sc
	}
	sc.limits = 1
	if name == "nolimits" {
		sc.limits = -1
	}
}

// scriptsFor returns the scripts node to attach a superscript (sup) or
// subscript to, wrapping the last atom if needed.
func (p *parser) scriptsFor(out *[]node, sup bool) *nScripts {
	n := len(*out)
	if n == 0 {
		sc := &nScripts{base: &nGroup{}}
		*out = append(*out, sc)
		return sc
	}
	if sc, ok := (*out)[n-1].(*nScripts); ok && ((sup && sc.sup == nil) || (!sup && sc.sub == nil)) {
		return sc
	}
	sc := &nScripts{base: (*out)[n-1]}
	(*out)[n-1] = sc
	return sc
}

// arg parses a macro argument: a braced group or a single atom.
func (p *parser) arg() node {
	p.skipSpace()
	switch {
	case p.eof(), p.s[p.i] == '}':
		return &nGroup{}
	case p.s[p.i] == '{':
		return p.group()
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
		p.i++
		return string(p.s[p.i-1])
	}
	start := p.i + 1
	end := p.matching(start)
	p.i = min(end+1, len(p.s))
	return string(p.s[start:end])
}

// matching returns the position of the '}' closing the group whose content
// starts at start, or the end of input.
func (p *parser) matching(start int) int {
	depth := 1
	for i := start; i < len(p.s); i++ {
		switch p.s[i] {
		case '\\':
			i++
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return len(p.s)
}

// optional returns the text of an optional [argument], if present.
func (p *parser) optional() (string, bool) {
	p.skipSpace()
	if p.eof() || p.s[p.i] != '[' {
		return "", false
	}
	start := p.i + 1
	end := p.closingBracket(start)
	p.i = min(end+1, len(p.s))
	return string(p.s[start:end]), true
}

// closingBracket returns the position of the ']' ending an optional
// argument (ignoring brackets inside braces), or the end of input.
func (p *parser) closingBracket(start int) int {
	depth := 0
	for i := start; i < len(p.s); i++ {
		switch p.s[i] {
		case '{':
			depth++
		case '}':
			depth--
		case ']':
			if depth == 0 {
				return i
			}
		}
	}
	return len(p.s)
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

// atom parses a group, a command or a single character.
func (p *parser) atom() node {
	switch p.s[p.i] {
	case '{':
		return p.group()
	case '\\':
		return p.command()
	case '~':
		p.i++
		return &nSpace{w: 0.33}
	}
	p.i++
	return p.char(p.s[p.i-1])
}

func (p *parser) char(c rune) node {
	switch {
	case isLetter(c) || (c >= 'α' && c <= 'ω'):
		return &nSym{r: c, cls: clsOrd, font: p.font}
	case c >= '0' && c <= '9':
		return &nSym{r: c, cls: clsOrd, font: p.digitFont()}
	case c == '-':
		c = '−'
	case c == '*':
		c = '∗'
	}
	cls, ok := charClasses[c]
	if !ok {
		cls = classOfRune(c)
	}
	return &nSym{r: c, cls: cls, font: fRoman}
}

// digitFont keeps digits upright unless a font like \mathbf is active.
func (p *parser) digitFont() fontStyle {
	if p.font == fItalic {
		return fRoman
	}
	return p.font
}

var runeClasses map[rune]class

// classOfRune gives typed Unicode symbols (≤, →, ∑ …) their usual class.
func classOfRune(r rune) class {
	if runeClasses == nil {
		runeClasses = buildRuneClasses()
	}
	if c, ok := runeClasses[r]; ok {
		return c
	}
	return clsOrd
}

func buildRuneClasses() map[rune]class {
	out := map[rune]class{}
	for _, s := range symbols {
		if c, seen := out[s.r]; seen && c != s.cls {
			out[s.r] = clsOrd // shared by \lvert, \rvert …: use the neutral class
			continue
		}
		out[s.r] = s.cls
	}
	return out
}

// parseLength converts a TeX length like "1em", "3mu" or "5pt" to em.
func parseLength(s string) float32 {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) && strings.ContainsRune("-.0123456789", rune(s[i])) {
		i++
	}
	v, err := strconv.ParseFloat(s[:i], 32)
	if err != nil {
		return 0
	}
	if f, ok := lengthUnits[strings.TrimSpace(s[i:])]; ok {
		v *= f
	}
	return float32(v)
}

// lengthUnits converts TeX units to em, assuming a 10pt font.
var lengthUnits = map[string]float64{"mu": 1.0 / 18, "pt": 0.1, "ex": 0.43, "cm": 2.85, "mm": 0.285}

// delim reads a delimiter after \left, \right, \big… (0 is the null
// delimiter ".").
func (p *parser) delim() rune {
	p.skipSpace()
	if p.eof() {
		return 0
	}
	if p.s[p.i] == '\\' {
		return delimiterByName(p.readCmd())
	}
	p.i++
	switch c := p.s[p.i-1]; c {
	case '.':
		return 0
	case '<':
		return '⟨'
	case '>':
		return '⟩'
	default:
		return c
	}
}

func delimiterByName(name string) rune {
	if r, ok := delimiterNames[name]; ok {
		return r
	}
	if s, ok := symbols[name]; ok {
		return s.r
	}
	return 0
}

// rows parses the body of an environment (or the whole input when env is
// ""), split into rows by \\ and cells by &.
func (p *parser) rows(env string) [][][]node {
	t := &table{}
	for !t.done {
		t.cell = append(t.cell, p.list()...)
		p.rowSeparator(t, env)
	}
	t.rows = append(t.rows, append(t.row, t.cell))
	// Drop a trailing empty row left by a final \\.
	if n := len(t.rows); n > 1 && len(t.rows[n-1]) == 1 && len(t.rows[n-1][0]) == 0 {
		t.rows = t.rows[:n-1]
	}
	return t.rows
}

// table collects the rows and cells of an environment while parsing.
type table struct {
	rows [][][]node
	row  [][]node
	cell []node
	done bool
}

// rowSeparator handles what ended a cell: &, \\, \end, or a stray token.
func (p *parser) rowSeparator(t *table, env string) {
	switch {
	case p.eof():
		t.done = true
	case p.s[p.i] == '&':
		p.i++
		t.row, t.cell = append(t.row, t.cell), nil
	case p.s[p.i] == '}':
		p.i++ // stray closing brace
	default:
		p.rowCommand(t, env)
	}
}

func (p *parser) rowCommand(t *table, env string) {
	switch name := p.readCmd(); name {
	case "\\", "cr":
		p.optional() // \\[2pt]
		t.rows = append(t.rows, append(t.row, t.cell))
		t.row, t.cell = nil, nil
	case "end":
		p.rawGroup()
		t.done = env != ""
	default: // stray \right or \middle
		t.cell = append(t.cell, &nError{s: "\\" + name})
	}
}
