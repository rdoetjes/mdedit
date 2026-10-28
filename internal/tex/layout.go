package tex

import (
	"strings"
	"sync"
)

// style is a TeX maths style: display, text, script or scriptscript.
type style int8

const (
	sD style = iota
	sT
	sS
	sSS
)

func (s style) scale() float32 {
	switch s {
	case sS:
		return 0.7
	case sSS:
		return 0.5
	}
	return 1
}

func (s style) script() style {
	if s <= sT {
		return sS
	}
	return sSS
}

func (s style) fraction() style {
	switch s {
	case sD:
		return sT
	case sT:
		return sS
	}
	return sSS
}

// Font parameters in em, approximating STIX Two Math's MATH table.
const (
	axisHeight  = 0.258
	ruleWidth   = 0.068
	xHeight     = 0.473
	scriptSpace = 0.05
	nullDelim   = 0.12
)

type ItemKind uint8

const (
	ItemGlyph ItemKind = iota
	ItemRule
	ItemPoly
)

// Item is a positioned element of a formula, in em with y pointing down and
// the origin on the baseline at the left of the formula.
type Item struct {
	Kind ItemKind
	// Glyph: X, Y is the glyph origin. Rule: X, Y is the top-left corner
	// and W, H the size.
	X, Y, W, H float32
	Face       uint8
	Glyph      uint16
	Size       float32 // glyph em size
	XScale     float32
	YScale     float32
	Points     [][2]float32 // polygon
	Error      bool         // part of an error message; drawn in red
}

// Box is a laid-out formula: W wide, extending H above and D below the
// baseline.
type Box struct {
	W, H, D float32
	Items   []Item
}

// place copies o's items into b, moved by (dx, dy).
func (b *Box) place(o *Box, dx, dy float32) {
	for _, it := range o.Items {
		it.X += dx
		it.Y += dy
		if it.Kind == ItemPoly {
			it.Points = movedPoints(it.Points, dx, dy)
		}
		b.Items = append(b.Items, it)
	}
}

func movedPoints(pts [][2]float32, dx, dy float32) [][2]float32 {
	out := make([][2]float32, len(pts))
	for i, p := range pts {
		out[i] = [2]float32{p[0] + dx, p[1] + dy}
	}
	return out
}

// hlist builds a box from boxes placed side by side.
type hlist struct {
	b Box
}

func (h *hlist) add(o *Box, dy float32) {
	h.b.place(o, h.b.W, dy)
	h.b.W += o.W
	h.b.H = max(h.b.H, o.H-dy)
	h.b.D = max(h.b.D, o.D+dy)
}

func (h *hlist) space(w float32) { h.b.W += w }

var (
	cacheMu sync.Mutex
	cache   = map[cacheKey]*Box{}
)

type cacheKey struct {
	src     string
	display bool
}

const maxCache = 2000

// Layout typesets TeX source in display or text (inline) style. Errors don't
// stop layout: unknown commands are shown in red.
func Layout(src string, display bool) *Box {
	k := cacheKey{src, display}
	cacheMu.Lock()
	b, ok := cache[k]
	cacheMu.Unlock()
	if ok {
		return b
	}
	b = layoutSource(src, display)
	cacheMu.Lock()
	if len(cache) > maxCache {
		cache = map[cacheKey]*Box{}
	}
	cache[k] = b
	cacheMu.Unlock()
	return b
}

func layoutSource(src string, display bool) *Box {
	st := sT
	if display {
		st = sD
	}
	rows := parse(src)
	if len(rows) == 1 && len(rows[0]) == 1 {
		return list(rows[0][0], st)
	}
	// Several lines: aligned if they have & alignment points.
	env := "gathered"
	for _, r := range rows {
		if len(r) > 1 {
			env = "aligned"
		}
	}
	b, _ := array(&nArray{env: env, rows: rows}, st)
	return b
}

// spacing between atom classes in mu (1/18 em); negative values apply only
// in display and text style.
var spacingTable = [8][8]int8{
	/* ord   */ {0, 3, -4, -5, 0, 0, 0, -3},
	/* op    */ {3, 3, 0, -5, 0, 0, 0, -3},
	/* bin   */ {-4, -4, 0, 0, -4, 0, 0, -4},
	/* rel   */ {-5, -5, 0, 0, -5, 0, 0, -5},
	/* open  */ {0, 0, 0, 0, 0, 0, 0, 0},
	/* close */ {0, 3, -4, -5, 0, 0, 0, -3},
	/* punct */ {-3, -3, 0, -3, -3, -3, -3, -3},
	/* inner */ {-3, 3, -4, -5, -3, 0, -3, -3},
}

func spacing(a, b class, st style) float32 {
	v := spacingTable[a][b]
	if v < 0 {
		if st >= sS {
			return 0
		}
		v = -v
	}
	return float32(v) / 18 * st.scale()
}

type atom struct {
	b   *Box
	cls class
}

// list lays out a sequence of nodes with TeX's inter-atom spacing.
func list(nodes []node, st style) *Box {
	atoms, cur := atomsOf(nodes, st)
	fixBinaries(atoms)
	return joinAtoms(atoms, cur)
}

// atomsOf lays out each node; style switches apply to the nodes after
// them. It also returns the style in effect at the end.
func atomsOf(nodes []node, st style) ([]atom, style) {
	var atoms []atom
	for _, n := range nodes {
		if s, ok := n.(*nStyle); ok {
			st = s.st
			continue
		}
		b, c := layoutNode(n, st)
		atoms = append(atoms, atom{b, c})
	}
	return atoms, st
}

// fixBinaries makes binary operators without a left operand (or followed
// by a relation or closing symbol) ordinary, as in a unary minus.
func fixBinaries(atoms []atom) {
	prev := clsOpen
	for i := range atoms {
		c := atoms[i].cls
		if c == clsNone {
			continue
		}
		if c == clsBin && unaryContext(prev, nextClass(atoms, i)) {
			atoms[i].cls = clsOrd
			c = clsOrd
		}
		prev = c
	}
}

// nextClass is the class of the next spacing atom after i.
func nextClass(atoms []atom, i int) class {
	for _, a := range atoms[i+1:] {
		if a.cls != clsNone {
			return a.cls
		}
	}
	return clsClose
}

func unaryContext(prev, next class) bool {
	switch prev {
	case clsBin, clsOp, clsRel, clsOpen, clsPunct:
		return true
	}
	return next == clsRel || next == clsClose || next == clsPunct
}

func joinAtoms(atoms []atom, st style) *Box {
	var h hlist
	last := class(-1)
	for _, a := range atoms {
		if a.cls != clsNone {
			if last >= 0 {
				h.space(spacing(last, a.cls, st))
			}
			last = a.cls
		}
		h.add(a.b, 0)
	}
	return &h.b
}

func glyphItem(face uint8, g uint16, size, yscale, y float32) Item {
	return Item{Kind: ItemGlyph, Face: face, Glyph: g, Size: size, XScale: 1, YScale: yscale, Y: y}
}

func glyphBox(r rune, size float32, errText bool) *Box {
	face, g, ok := lookup(r)
	if !ok {
		face, g, ok = lookup('?')
	}
	if !ok {
		return &Box{W: 0.5 * size}
	}
	m := glyphInfo(face, g)
	it := glyphItem(face, g, size, 1, 0)
	it.Error = errText
	return &Box{W: m.adv * size, H: max(0, -m.minY) * size, D: max(0, m.maxY) * size, Items: []Item{it}}
}

// axisGlyph returns a glyph of em size gs centred vertically on the maths
// axis, which is at height axis.
func axisGlyph(face uint8, g uint16, m glyphMetrics, gs, axis float32) *Box {
	dy := -axis - (m.minY+m.maxY)/2*gs
	return &Box{W: m.adv * gs, H: -m.minY*gs - dy, D: m.maxY*gs + dy, Items: []Item{glyphItem(face, g, gs, 1, dy)}}
}

// symBox lays out a character in a mathematical alphabet, falling back to
// the plain character if the font lacks the styled form.
func symBox(r rune, f fontStyle, size float32) *Box {
	if s := styled(r, f); s != r {
		if _, _, ok := lookup(s); ok {
			return glyphBox(s, size, false)
		}
	}
	return glyphBox(r, size, false)
}

func textBox(s string, f fontStyle, size float32, errText bool) *Box {
	var h hlist
	for _, r := range s {
		switch {
		case r == ' ':
			h.space(0.25 * size)
		case errText:
			h.add(glyphBox(r, size, true), 0)
		default:
			h.add(symBox(r, f, size), 0)
		}
	}
	return &h.b
}

func layoutNode(n node, st style) (*Box, class) {
	size := st.scale()
	switch n := n.(type) {
	case *nSym:
		return symBox(n.r, n.font, size), n.cls
	case *nGroup:
		return list(n.items, st), clsOrd
	case *nOp:
		b, _ := opBox(n, st)
		return b, clsOp
	case *nScripts:
		return scripts(n, st)
	case *nFrac:
		return fraction(n, st), clsInner
	case *nSqrt:
		return sqrt(n, st), clsOrd
	case *nLeftRight:
		return leftRight(n, st), clsInner
	case *nAccent:
		return accent(n, st)
	case *nSpace:
		return &Box{W: n.w * size}, clsNone
	case *nText:
		return textBox(n.s, n.font, size, false), clsOrd
	case *nArray:
		return array(n, st)
	case *nBig:
		return delimiter(n.r, n.size*size, size), n.cls
	case *nError:
		return textBox(n.s, fRoman, size, true), clsOrd
	}
	return &Box{}, clsNone
}

// opBox lays out a large operator or operator name. For large operators
// it reports whether the glyph is an integral (whose scripts sit closer).
func opBox(n *nOp, st style) (*Box, bool) {
	size := st.scale()
	if n.name != "" {
		return textBox(n.name, fRoman, size, false), false
	}
	integral := strings.ContainsRune("∫∬∭∮∯", n.r)
	face, g, ok := lookup(n.r)
	if !ok {
		return glyphBox(n.r, size, false), integral
	}
	return axisGlyph(face, g, glyphInfo(face, g), size*opScale(st, integral), axisHeight*size), integral
}

// opScale enlarges large operators in display style.
func opScale(st style, integral bool) float32 {
	switch {
	case st != sD:
		return 1
	case integral:
		return 1.8
	}
	return 1.4
}

// quad returns a polygon for a straight stroke of width w.
func quad(x1, y1, x2, y2, w float32) Item {
	dx, dy := x2-x1, y2-y1
	l := sqrt32(dx*dx + dy*dy)
	if l == 0 {
		l = 1
	}
	nx, ny := -dy/l*w/2, dx/l*w/2
	return Item{Kind: ItemPoly, Points: [][2]float32{
		{x1 + nx, y1 + ny}, {x2 + nx, y2 + ny}, {x2 - nx, y2 - ny}, {x1 - nx, y1 - ny},
	}}
}

func sqrt32(v float32) float32 {
	// Newton's method is plenty for drawing.
	if v <= 0 {
		return 0
	}
	x := v
	for i := 0; i < 20; i++ {
		x = 0.5 * (x + v/x)
	}
	return x
}
