package tex

import "strings"

// commandHandler parses the arguments of a control sequence.
type commandHandler func(p *parser, name string) node

// handlers maps control sequences with arguments to their parsers. It is
// filled in init because the handlers refer back to the parser.
var handlers map[string]commandHandler

func init() {
	handlers = map[string]commandHandler{
		"frac": (*parser).frac, "dfrac": (*parser).frac, "tfrac": (*parser).frac, "cfrac": (*parser).frac,
		"binom": (*parser).binom, "dbinom": (*parser).binom, "tbinom": (*parser).binom,
		"sqrt": (*parser).sqrt,
		"left": (*parser).leftRight,
		"not":  (*parser).not,
		"text": (*parser).text, "textrm": (*parser).text, "textnormal": (*parser).text, "mbox": (*parser).text,
		"hbox": (*parser).text, "textup": (*parser).text, "textsf": (*parser).text, "texttt": (*parser).text,
		"textit": (*parser).text, "emph": (*parser).text, "textbf": (*parser).text,
		"operatorname": (*parser).operatorName, "operatornamewithlimits": (*parser).operatorName,
		"displaystyle": (*parser).styleSwitch, "textstyle": (*parser).styleSwitch,
		"scriptstyle": (*parser).styleSwitch, "scriptscriptstyle": (*parser).styleSwitch,
		"overset": (*parser).overset, "stackrel": (*parser).overset, "underset": (*parser).underset,
		"begin": (*parser).environment,
		"pmod":  (*parser).pmod, "bmod": (*parser).bmod, "mod": (*parser).mod,
		"color": (*parser).color, "textcolor": (*parser).textColor,
		"tag": (*parser).tag, "tag*": (*parser).tag,
		"hspace": (*parser).hspace, "kern": (*parser).hspace, "mkern": (*parser).hspace,
		"mskip": (*parser).hspace, "hskip": (*parser).hspace,
		"label": (*parser).label, "nonumber": (*parser).ignore, "notag": (*parser).ignore,
		"right": (*parser).stray, "middle": (*parser).stray, "end": (*parser).stray,
	}
	for _, size := range []string{"big", "Big", "bigg", "Bigg"} {
		for _, suffix := range []string{"", "l", "r", "m"} {
			handlers[size+suffix] = (*parser).big
		}
	}
	for _, kind := range []string{"bar", "overline", "underline", "vec", "overrightarrow", "overleftarrow",
		"overbrace", "underbrace", "boxed", "cancel", "phantom", "hphantom", "vphantom"} {
		handlers[kind] = (*parser).accent
	}
}

// command parses a control sequence and its arguments.
func (p *parser) command() node {
	name := p.readCmd()
	if n, ok := p.simpleCommand(name); ok {
		return n
	}
	if f, ok := fontCommands[name]; ok {
		return p.withFont(f)
	}
	if _, ok := accents[name]; ok {
		return p.accent(name)
	}
	if h, ok := handlers[name]; ok {
		return h(p, name)
	}
	if name == "" {
		return nil
	}
	return &nError{s: "\\" + name}
}

// simpleCommand handles commands without arguments: spaces, symbols, big
// operators and function names.
func (p *parser) simpleCommand(name string) (node, bool) {
	if w, ok := spaces[name]; ok {
		return &nSpace{w: w}, true
	}
	if s, ok := symbols[name]; ok {
		return p.symbol(s), true
	}
	if op, ok := bigOps[name]; ok {
		return &nOp{r: op.r, limits: op.limits}, true
	}
	if limits, ok := functions[name]; ok {
		return &nOp{name: functionName(name), limits: limits}, true
	}
	return nil, false
}

// symbol keeps lowercase Greek in the current (italic) font; other symbols
// are upright.
func (p *parser) symbol(s symbol) node {
	f := fRoman
	if s.r >= 'α' && s.r <= 'ω' || italicGreekVariants[s.r] != 0 {
		f = p.font
	}
	return &nSym{r: s.r, cls: s.cls, font: f}
}

func functionName(name string) string {
	if dn, ok := functionNames[name]; ok {
		return dn
	}
	return name
}

func (p *parser) withFont(f fontStyle) node {
	old := p.font
	p.font = f
	body := p.arg()
	p.font = old
	return body
}

var fracStyles = map[string]int8{"frac": -1, "dfrac": 0, "cfrac": 0, "tfrac": 1, "binom": -1, "dbinom": 0, "tbinom": 1}

func (p *parser) frac(name string) node {
	return &nFrac{num: p.arg(), den: p.arg(), bar: true, style: fracStyles[name]}
}

func (p *parser) binom(name string) node {
	return &nFrac{num: p.arg(), den: p.arg(), style: fracStyles[name], left: '(', right: ')'}
}

func (p *parser) sqrt(string) node {
	var index node
	if s, ok := p.optional(); ok {
		index = p.sub(s)
	}
	return &nSqrt{index: index, body: p.arg()}
}

// leftRight parses \left … \middle … \right.
func (p *parser) leftRight(string) node {
	lr := &nLeftRight{delims: []rune{p.delim()}}
	for {
		lr.parts = append(lr.parts, p.list())
		if p.peekCmd() != "middle" {
			break
		}
		p.readCmd()
		lr.delims = append(lr.delims, p.delim())
	}
	lr.delims = append(lr.delims, p.rightDelim())
	return lr
}

// rightDelim reads the delimiter after \right; an unclosed \left (or a stray
// terminator) gets the null delimiter.
func (p *parser) rightDelim() rune {
	if p.peekCmd() == "right" {
		p.readCmd()
		return p.delim()
	}
	if !p.eof() && p.s[p.i] == '}' {
		p.i++
	}
	return 0
}

var bigSizes = map[string]float32{"big": 1.2, "Big": 1.8, "bigg": 2.4, "Bigg": 3.0}

func (p *parser) big(name string) node {
	cls := clsOrd
	switch name[len(name)-1] {
	case 'l':
		cls = clsOpen
	case 'r':
		cls = clsClose
	case 'm':
		cls = clsRel
	}
	return &nBig{r: p.delim(), size: bigSizes[strings.TrimRight(name, "lrm")], cls: cls}
}

func (p *parser) accent(name string) node { return &nAccent{kind: name, body: p.arg()} }

// not negates the next symbol, using a precomposed character if there is
// one.
func (p *parser) not(string) node {
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
}

var negations = map[rune]rune{
	'=': '≠', '∈': '∉', '<': '≮', '>': '≯', '≤': '≰', '≥': '≱', '⊂': '⊄', '⊃': '⊅',
	'⊆': '⊈', '⊇': '⊉', '∼': '≁', '≡': '≢', '≈': '≉', '∃': '∄', '∣': '∤', '∥': '∦', '≅': '≇',
}

var textFonts = map[string]fontStyle{"textit": fItalic, "emph": fItalic, "textbf": fBold}

func (p *parser) text(name string) node {
	f, ok := textFonts[name]
	if !ok {
		f = fRoman
	}
	return &nText{s: p.rawGroup(), font: f}
}

func (p *parser) operatorName(name string) node {
	limits := name == "operatornamewithlimits"
	if !p.eof() && p.s[p.i] == '*' {
		p.i++
		limits = true
	}
	return &nOp{name: p.rawGroup(), limits: limits}
}

var styleSwitches = map[string]style{"displaystyle": sD, "textstyle": sT, "scriptstyle": sS, "scriptscriptstyle": sSS}

func (p *parser) styleSwitch(name string) node { return &nStyle{st: styleSwitches[name]} }

func (p *parser) overset(string) node {
	over := p.arg()
	return &nScripts{base: p.arg(), sup: over, limits: 1}
}

func (p *parser) underset(string) node {
	under := p.arg()
	return &nScripts{base: p.arg(), sub: under, limits: 1}
}

func (p *parser) environment(string) node {
	env := strings.TrimSpace(p.rawGroup())
	a := &nArray{env: strings.TrimSuffix(env, "*")}
	if a.env == "array" {
		a.cols = p.rawGroup()
	}
	a.rows = p.rows(env)
	return a
}

func symNode(r rune, cls class) node { return &nSym{r: r, cls: cls, font: fRoman} }

func (p *parser) pmod(string) node {
	arg := p.arg()
	return &nGroup{items: []node{&nSpace{w: 1}, symNode('(', clsOpen), &nOp{name: "mod"},
		&nSpace{w: 6.0 / 18}, arg, symNode(')', clsClose)}}
}

func (p *parser) bmod(string) node {
	return &nGroup{items: []node{&nSpace{w: 4.0 / 18}, &nOp{name: "mod"}, &nSpace{w: 4.0 / 18}}}
}

func (p *parser) mod(string) node {
	return &nGroup{items: []node{&nSpace{w: 1}, &nOp{name: "mod"}, &nSpace{w: 6.0 / 18}}}
}

// color ignores \color{…}: everything is drawn in the text colour.
func (p *parser) color(string) node {
	p.rawGroup()
	return nil
}

func (p *parser) textColor(string) node {
	p.rawGroup()
	return p.arg()
}

func (p *parser) tag(string) node {
	return &nGroup{items: []node{&nSpace{w: 2}, &nText{s: "(" + p.rawGroup() + ")", font: fRoman}}}
}

func (p *parser) hspace(string) node { return &nSpace{w: parseLength(p.rawGroup())} }

func (p *parser) label(string) node {
	p.rawGroup()
	return nil
}

func (p *parser) ignore(string) node { return nil }

// stray reports \right, \middle or \end without their opening command.
func (p *parser) stray(name string) node { return &nError{s: "\\" + name} }
