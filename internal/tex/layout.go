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

func (b *Box) place(o *Box, dx, dy float32) {
	for _, it := range o.Items {
		it.X += dx
		it.Y += dy
		if it.Kind == ItemPoly {
			pts := make([][2]float32, len(it.Points))
			for i, p := range it.Points {
				pts[i] = [2]float32{p[0] + dx, p[1] + dy}
			}
			it.Points = pts
		}
		b.Items = append(b.Items, it)
	}
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

// Layout typesets TeX source in display or text (inline) style. Errors don't
// stop layout: unknown commands are shown in red.
func Layout(src string, display bool) *Box {
	k := cacheKey{src, display}
	cacheMu.Lock()
	if b, ok := cache[k]; ok {
		cacheMu.Unlock()
		return b
	}
	cacheMu.Unlock()

	st := sT
	if display {
		st = sD
	}
	rows := parse(src)
	var b *Box
	if len(rows) == 1 && len(rows[0]) == 1 {
		b = list(rows[0][0], st)
	} else {
		env := "gathered"
		for _, r := range rows {
			if len(r) > 1 {
				env = "aligned"
			}
		}
		b, _ = array(&nArray{env: env, rows: rows}, st)
	}

	cacheMu.Lock()
	if len(cache) > 2000 {
		cache = map[cacheKey]*Box{}
	}
	cache[k] = b
	cacheMu.Unlock()
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
	var atoms []atom
	cur := st
	for _, n := range nodes {
		if s, ok := n.(*nStyle); ok {
			cur = s.st
			continue
		}
		b, c := layoutNode(n, cur)
		atoms = append(atoms, atom{b, c})
	}
	// A binary operator without a left operand (or followed by a relation
	// or closing symbol) is ordinary, as in a unary minus.
	prev := clsOpen
	for i := range atoms {
		c := atoms[i].cls
		if c == clsNone {
			continue
		}
		if c == clsBin {
			next := clsClose
			for j := i + 1; j < len(atoms); j++ {
				if atoms[j].cls != clsNone {
					next = atoms[j].cls
					break
				}
			}
			if prev == clsBin || prev == clsOp || prev == clsRel || prev == clsOpen || prev == clsPunct ||
				next == clsRel || next == clsClose || next == clsPunct {
				atoms[i].cls = clsOrd
				c = clsOrd
			}
		}
		prev = c
	}
	var h hlist
	last := class(-1)
	for _, a := range atoms {
		if a.cls != clsNone {
			if last >= 0 {
				h.space(spacing(last, a.cls, cur))
			}
			last = a.cls
		}
		h.add(a.b, 0)
	}
	return &h.b
}

func glyphBox(r rune, size float32, errText bool) *Box {
	face, g, ok := lookup(r)
	if !ok {
		face, g, ok = lookup('?')
		if !ok {
			return &Box{W: 0.5 * size}
		}
	}
	m := glyphInfo(face, g)
	return &Box{
		W: m.adv * size, H: max(0, -m.minY) * size, D: max(0, m.maxY) * size,
		Items: []Item{{Kind: ItemGlyph, Face: face, Glyph: g, Size: size, XScale: 1, YScale: 1, Error: errText}},
	}
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
		if r == ' ' {
			h.space(0.25 * size)
			continue
		}
		var b *Box
		if errText {
			b = glyphBox(r, size, true)
		} else {
			b = symBox(r, f, size)
		}
		h.add(b, 0)
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
	scale := float32(1)
	if st == sD {
		scale = 1.4
		if integral {
			scale = 1.8
		}
	}
	face, g, ok := lookup(n.r)
	if !ok {
		return glyphBox(n.r, size, false), integral
	}
	m := glyphInfo(face, g)
	gs := size * scale
	// Centre the operator on the maths axis.
	dy := -axisHeight*size - (m.minY+m.maxY)/2*gs
	return &Box{
		W: m.adv * gs, H: -m.minY*gs - dy, D: m.maxY*gs + dy,
		Items: []Item{{Kind: ItemGlyph, Face: face, Glyph: g, Size: gs, XScale: 1, YScale: 1, Y: dy}},
	}, integral
}

func scripts(n *nScripts, st style) (*Box, class) {
	size := st.scale()
	var base *Box
	cls := clsOrd
	integral := false
	limits := n.limits == 1
	switch b := n.base.(type) {
	case *nOp:
		base, integral = opBox(b, st)
		cls = clsOp
		if b.limits && st == sD && n.limits != -1 {
			limits = true
		}
	case *nAccent:
		if b.kind == "overbrace" || b.kind == "underbrace" {
			limits = n.limits != -1
		}
		base, cls = layoutNode(b, st)
	default:
		base, cls = layoutNode(n.base, st)
	}
	sst := st.script()
	ss := sst.scale()
	var sup, sub *Box
	if n.sup != nil {
		sup, _ = layoutNode(n.sup, sst)
	}
	if n.sub != nil {
		sub, _ = layoutNode(n.sub, sst)
	}

	if limits {
		w := base.W
		if sup != nil {
			w = max(w, sup.W)
		}
		if sub != nil {
			w = max(w, sub.W)
		}
		out := &Box{W: w, H: base.H, D: base.D}
		out.place(base, (w-base.W)/2, 0)
		gap := 0.12 * size
		if sup != nil {
			dy := -(base.H + gap + sup.D)
			out.place(sup, (w-sup.W)/2, dy)
			out.H = base.H + gap + sup.D + sup.H
		}
		if sub != nil {
			dy := base.D + gap + sub.H
			out.place(sub, (w-sub.W)/2, dy)
			out.D = base.D + gap + sub.H + sub.D
		}
		return out, cls
	}

	// Superscript and subscript shifts, following TeX's rules 18a–f.
	_, isChar := n.base.(*nSym)
	u, v := float32(0), float32(0)
	if !isChar {
		u = base.H - 0.25*ss
		v = base.D + 0.05*ss
	}
	if sup != nil {
		minSup := float32(0.363)
		if st == sD {
			minSup = 0.413
		}
		u = max(u, minSup*size, sup.D+xHeight*size/4)
	}
	if sub != nil {
		v = max(v, 0.15*size, sub.H-0.8*xHeight*size)
	}
	if sup != nil && sub != nil {
		v = max(v, 0.247*size)
		if gap := (u - sup.D) - (sub.H - v); gap < 4*ruleWidth*size {
			v += 4*ruleWidth*size - gap
			if psi := 0.8*xHeight*size - (u - sup.D); psi > 0 {
				u += psi
				v -= psi
			}
		}
	}
	supX, subX := base.W, base.W
	if integral {
		subX = base.W * 0.65
		supX = base.W * 1.02
	}
	out := &Box{W: base.W, H: base.H, D: base.D}
	out.place(base, 0, 0)
	if sup != nil {
		out.place(sup, supX, -u)
		out.W = max(out.W, supX+sup.W)
		out.H = max(out.H, u+sup.H)
		out.D = max(out.D, sup.D-u)
	}
	if sub != nil {
		out.place(sub, subX, v)
		out.W = max(out.W, subX+sub.W)
		out.D = max(out.D, v+sub.D)
		out.H = max(out.H, sub.H-v)
	}
	out.W += scriptSpace * size
	return out, cls
}

func fraction(n *nFrac, st style) *Box {
	fst := st
	switch n.style {
	case 0:
		fst = sD
	case 1:
		fst = max(st, sT)
	}
	size := fst.scale()
	num, _ := layoutNode(n.num, fst.fraction())
	den, _ := layoutNode(n.den, fst.fraction())
	rule := ruleWidth * size
	if !n.bar {
		rule = 0
	}
	axis := axisHeight * size
	var numShift, denShift, gap float32
	if fst == sD {
		numShift, denShift, gap = 0.677*size, 0.686*size, 3*ruleWidth*size
	} else {
		numShift, denShift, gap = 0.394*size, 0.345*size, ruleWidth*size
	}
	if !n.bar {
		gap *= 2
	}
	numShift = max(numShift, axis+rule/2+gap+num.D)
	denShift = max(denShift, -axis+rule/2+gap+den.H)

	pad := nullDelim * size
	if !n.bar {
		pad = 0
	}
	w := max(num.W, den.W) + 2*pad
	out := &Box{W: w, H: numShift + num.H, D: denShift + den.D}
	out.place(num, (w-num.W)/2, -numShift)
	out.place(den, (w-den.W)/2, denShift)
	if n.bar {
		out.Items = append(out.Items, Item{Kind: ItemRule, X: pad / 2, Y: -axis - rule/2, W: w - pad, H: rule})
	}
	if n.left != 0 || n.right != 0 {
		return wrapDelims(out, n.left, n.right, size)
	}
	return out
}

// wrapDelims puts delimiters sized to the box around it.
func wrapDelims(b *Box, left, right rune, size float32) *Box {
	total := delimHeight(b, size)
	var h hlist
	h.add(delimiter(left, total, size), 0)
	h.add(b, 0)
	h.add(delimiter(right, total, size), 0)
	return &h.b
}

// delimHeight is the delimiter size needed to cover a box, by TeX's rule.
func delimHeight(b *Box, size float32) float32 {
	axis := axisHeight * size
	half := max(b.H-axis, b.D+axis)
	return max(2*half*0.901, 2*half-0.5*size)
}

// Stretchable delimiters built from the Unicode bracket pieces:
// top, extender, bottom and middle.
var delimPieces = map[rune][4]rune{
	'(': {'⎛', '⎜', '⎝', 0}, ')': {'⎞', '⎟', '⎠', 0},
	'[': {'⎡', '⎢', '⎣', 0}, ']': {'⎤', '⎥', '⎦', 0},
	'{': {'⎧', '⎪', '⎩', '⎨'}, '}': {'⎫', '⎪', '⎭', '⎬'},
	'⌈': {'⎡', '⎢', '⎢', 0}, '⌉': {'⎤', '⎥', '⎥', 0},
	'⌊': {'⎢', '⎢', '⎣', 0}, '⌋': {'⎥', '⎥', '⎦', 0},
}

// delimiter returns delimiter r with the given total height, centred on the
// maths axis. r == 0 is the null delimiter.
func delimiter(r rune, total, size float32) *Box {
	if r == 0 {
		return &Box{W: nullDelim * size}
	}
	face, g, ok := lookup(r)
	if !ok {
		return glyphBox(r, size, false)
	}
	m := glyphInfo(face, g)
	axis := axisHeight * size
	top, bot := -(total/2 + axis), total/2-axis
	nat := (m.maxY - m.minY) * size
	if total <= nat*1.05 {
		// The normal glyph is big enough: centre it on the axis.
		dy := -axis - (m.minY+m.maxY)/2*size
		return &Box{W: m.adv * size, H: -m.minY*size - dy, D: m.maxY*size + dy,
			Items: []Item{{Kind: ItemGlyph, Face: face, Glyph: g, Size: size, XScale: 1, YScale: 1, Y: dy}}}
	}
	out := &Box{W: m.adv * size, H: -top, D: bot}
	pieces, ok := delimPieces[r]
	if !ok || total < nat*1.4 {
		// Stretch the glyph vertically.
		ys := total / nat
		dy := top - m.minY*size*ys
		out.Items = []Item{{Kind: ItemGlyph, Face: face, Glyph: g, Size: size, XScale: 1, YScale: ys, Y: dy}}
		return out
	}
	// Assemble from pieces: ends at the top and bottom, extenders between.
	piece := func(r rune) (uint8, uint16, glyphMetrics) {
		f, g, _ := lookup(r)
		return f, g, glyphInfo(f, g)
	}
	tf, tg, tm := piece(pieces[0])
	ef, eg, em := piece(pieces[1])
	bf, bg, bm := piece(pieces[2])
	out.W = max(tm.adv, bm.adv, em.adv) * size
	add := func(f uint8, g uint16, y, ys float32) {
		out.Items = append(out.Items, Item{Kind: ItemGlyph, Face: f, Glyph: g, Size: size, XScale: 1, YScale: ys, Y: y})
	}
	add(tf, tg, top-tm.minY*size, 1)
	add(bf, bg, bot-bm.maxY*size, 1)
	tBottom := top + (tm.maxY-tm.minY)*size
	bTop := bot - (bm.maxY-bm.minY)*size
	extend := func(from, to float32) {
		overlap := 0.02 * size
		span := to - from + 2*overlap
		eh := (em.maxY - em.minY) * size
		if span <= 0 || eh <= 0 {
			return
		}
		ys := span / eh
		add(ef, eg, from-overlap-em.minY*size*ys, ys)
	}
	if pieces[3] != 0 {
		mf, mg, mm := piece(pieces[3])
		mdy := -axis - (mm.minY+mm.maxY)/2*size
		add(mf, mg, mdy, 1)
		extend(tBottom, mdy+mm.minY*size)
		extend(mdy+mm.maxY*size, bTop)
	} else {
		extend(tBottom, bTop)
	}
	return out
}

func leftRight(n *nLeftRight, st style) *Box {
	size := st.scale()
	parts := make([]*Box, len(n.parts))
	var hh, dd float32
	for i, p := range n.parts {
		parts[i] = list(p, st)
		hh = max(hh, parts[i].H)
		dd = max(dd, parts[i].D)
	}
	total := delimHeight(&Box{H: hh, D: dd}, size)
	var h hlist
	for i, d := range n.delims {
		h.add(delimiter(d, total, size), 0)
		if i < len(parts) {
			h.add(parts[i], 0)
		}
	}
	return &h.b
}

// quad returns a polygon for a straight stroke of width w.
func quad(x1, y1, x2, y2, w float32) Item {
	dx, dy := x2-x1, y2-y1
	l := float32(sqrt32(dx*dx + dy*dy))
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

func sqrt(n *nSqrt, st style) *Box {
	size := st.scale()
	body, _ := layoutNode(n.body, st)
	rule := ruleWidth * size
	phi := rule
	if st == sD {
		phi = xHeight * size
	}
	gap := rule + phi/4
	innerTop := body.H + gap + rule // top of the vinculum
	bottom := max(body.D, 0.1*size)
	total := innerTop + bottom
	hs := min(total, 1.1*size)
	sw := min(0.5*size+0.1*total, 1.1*size)

	// The radical sign: a short rising tick, a heavy down-stroke and a
	// long thin up-stroke meeting the vinculum.
	yTop := -innerTop + rule/2
	ax, ay := 0.0*size, bottom-0.48*hs
	bx, by := 0.12*size, ay-0.07*size
	cx, cy := 0.28*size+0.08*(sw-0.6*size), bottom
	dx := sw

	var h hlist
	sign := &Box{W: sw, H: innerTop, D: bottom}
	sign.Items = append(sign.Items,
		quad(ax, ay, bx, by, 0.045*size),
		quad(bx, by, cx, cy, 0.095*size),
		quad(cx, cy, dx, yTop, 0.045*size),
	)

	shift := float32(0)
	if n.index != nil {
		idx, _ := layoutNode(n.index, sSS)
		// Raise the index over the tick, moving the sign right if needed.
		room := bx + 0.06*size
		if idx.W > room {
			shift = idx.W - room
		}
		var ib Box
		ib.place(idx, shift+room-idx.W, bottom-0.6*total-idx.D)
		ib.W = 0
		ib.H = 0.6*total - bottom + idx.D + idx.H
		h.add(&ib, 0)
		h.space(shift)
	}
	h.add(sign, 0)
	vincW := body.W + 0.1*size
	vinc := &Box{W: 0, H: innerTop, Items: []Item{{Kind: ItemRule, X: -0.02 * size, Y: -innerTop, W: vincW + 0.02*size, H: rule}}}
	h.add(vinc, 0)
	h.add(body, 0)
	h.space(0.1 * size)
	h.b.H = max(h.b.H, innerTop+rule)
	return &h.b
}

func accent(n *nAccent, st style) (*Box, class) {
	size := st.scale()
	body, cls := layoutNode(n.body, st)
	rule := ruleWidth * size
	out := &Box{W: body.W, H: body.H, D: body.D}
	out.place(body, 0, 0)
	_, single := n.body.(*nSym)
	skew := float32(0)
	if single {
		if s := n.body.(*nSym); s.font == fItalic && isLetter(s.r) {
			skew = 0.08 * size
		}
	}

	switch n.kind {
	case "overline", "bar":
		w, x := body.W, float32(0)
		if n.kind == "bar" && single {
			w = body.W * 0.8
			x = (body.W-w)/2 + skew
		}
		y := -(body.H + 3*rule + rule)
		out.Items = append(out.Items, Item{Kind: ItemRule, X: x, Y: y, W: w, H: rule})
		out.H = -y + rule
		return out, cls
	case "underline":
		out.Items = append(out.Items, Item{Kind: ItemRule, Y: body.D + 3*rule, W: body.W, H: rule})
		out.D = body.D + 5*rule
		return out, cls
	case "vec", "overrightarrow", "overleftarrow":
		w, x := body.W, float32(0)
		if n.kind == "vec" {
			w = max(0.45*size, body.W*0.7)
			x = (body.W-w)/2 + skew
		}
		y := -(body.H + 0.15*size)
		th := 0.045 * size
		head := 0.13 * size
		out.Items = append(out.Items, quad(x, y, x+w, y, th))
		if n.kind == "overleftarrow" {
			out.Items = append(out.Items, quad(x, y, x+head, y-head*0.7, th), quad(x, y, x+head, y+head*0.7, th))
		} else {
			out.Items = append(out.Items, quad(x+w, y, x+w-head, y-head*0.7, th), quad(x+w, y, x+w-head, y+head*0.7, th))
		}
		out.H = -y + head*0.8
		return out, cls
	case "overbrace", "underbrace":
		th := 0.05 * size
		hgt := 0.2 * size
		w := body.W
		var y0, y1 float32 // y0: ends of the brace, y1: its tip
		if n.kind == "overbrace" {
			y0 = -(body.H + 0.1*size)
			y1 = y0 - hgt
			out.H = -y1
		} else {
			y0 = body.D + 0.1*size
			y1 = y0 + hgt
			out.D = y1
		}
		ym := (y0 + y1) / 2
		out.Items = append(out.Items,
			quad(0, y0, hgt/2, ym, th), quad(hgt/2, ym, w/2-hgt/2, ym, th),
			quad(w/2-hgt/2, ym, w/2, y1, th), quad(w/2, y1, w/2+hgt/2, ym, th),
			quad(w/2+hgt/2, ym, w-hgt/2, ym, th), quad(w-hgt/2, ym, w, y0, th))
		return out, clsOp
	case "boxed":
		pad := 0.2 * size
		var b Box
		b.place(body, pad, 0)
		b.W, b.H, b.D = body.W+2*pad, body.H+pad, body.D+pad
		t := 0.04 * size
		b.Items = append(b.Items,
			Item{Kind: ItemRule, X: 0, Y: -b.H, W: b.W, H: t},
			Item{Kind: ItemRule, X: 0, Y: b.D - t, W: b.W, H: t},
			Item{Kind: ItemRule, X: 0, Y: -b.H, W: t, H: b.H + b.D},
			Item{Kind: ItemRule, X: b.W - t, Y: -b.H, W: t, H: b.H + b.D})
		return &b, cls
	case "cancel":
		out.Items = append(out.Items, quad(0, body.D, body.W, -body.H, 0.04*size))
		return out, cls
	case "not":
		out.Items = append(out.Items, quad(body.W*0.25, body.D+0.05*size, body.W*0.75, -body.H-0.05*size, 0.05*size))
		return out, cls
	case "phantom":
		return &Box{W: body.W, H: body.H, D: body.D}, cls
	case "hphantom":
		return &Box{W: body.W}, cls
	case "vphantom":
		return &Box{H: body.H, D: body.D}, cls
	}

	r := accents[n.kind]
	face, g, ok := lookup(r)
	if !ok {
		return out, cls
	}
	m := glyphInfo(face, g)
	inkW := (m.maxX - m.minX) * size
	xs := float32(1)
	if strings.HasPrefix(n.kind, "wide") && inkW > 0 {
		xs = max(1, body.W*0.9/inkW)
	}
	gap := 0.05 * size
	y := -(body.H + gap) - m.maxY*size
	x := (body.W-inkW*xs)/2 - m.minX*size*xs + skew
	out.Items = append(out.Items, Item{Kind: ItemGlyph, Face: face, Glyph: g, Size: size, XScale: xs, YScale: 1, X: x, Y: y})
	out.H = max(out.H, body.H+gap+(m.maxY-m.minY)*size)
	return out, cls
}

// array lays out matrices, cases and aligned equations.
func array(a *nArray, st style) (*Box, class) {
	cst := sT
	switch a.env {
	case "aligned", "align", "alignat", "gathered", "gather", "split", "multline", "equation", "dcases":
		if st == sD {
			cst = sD
		}
	case "smallmatrix":
		cst = sS
	}
	if st >= sS {
		cst = max(cst, st)
	}
	size := cst.scale()

	ncol := 0
	for _, r := range a.rows {
		ncol = max(ncol, len(r))
	}
	if ncol == 0 {
		return &Box{}, clsOrd
	}
	// Column alignment.
	align := make([]byte, ncol)
	for i := range align {
		align[i] = 'c'
	}
	colGap := make([]float32, ncol) // space after each column
	for i := range colGap {
		colGap[i] = 1.0 * size
	}
	aligned := false
	switch a.env {
	case "aligned", "align", "alignat", "split":
		aligned = true
		for i := range align {
			if i%2 == 0 {
				align[i] = 'r'
				colGap[i] = 0
			} else {
				align[i] = 'l'
				colGap[i] = 1.5 * size
			}
		}
	case "cases", "dcases":
		for i := range align {
			align[i] = 'l'
		}
	case "array":
		j := 0
		for _, c := range a.cols {
			if (c == 'l' || c == 'c' || c == 'r') && j < ncol {
				align[j] = byte(c)
				j++
			}
		}
	case "smallmatrix":
		for i := range colGap {
			colGap[i] = 0.5 * size
		}
	}

	cells := make([][]*Box, len(a.rows))
	widths := make([]float32, ncol)
	for i, r := range a.rows {
		cells[i] = make([]*Box, ncol)
		for j := 0; j < ncol; j++ {
			var nodes []node
			if j < len(r) {
				nodes = r[j]
			}
			if aligned && j%2 == 1 {
				// Give a leading relation its left spacing: "&= b".
				nodes = append([]node{&nGroup{}}, nodes...)
			}
			b := list(nodes, cst)
			cells[i][j] = b
			widths[j] = max(widths[j], b.W)
		}
	}
	rowGap := 0.25 * size
	if cst == sD {
		rowGap = 0.4 * size
	}
	out := &Box{}
	y := float32(0)
	for i := range cells {
		rh, rd := 0.7*size, 0.3*size
		for _, b := range cells[i] {
			rh = max(rh, b.H)
			rd = max(rd, b.D)
		}
		if i > 0 {
			y += rowGap
		}
		y += rh
		x := float32(0)
		for j, b := range cells[i] {
			dx := float32(0)
			switch align[j] {
			case 'c':
				dx = (widths[j] - b.W) / 2
			case 'r':
				dx = widths[j] - b.W
			}
			out.place(b, x+dx, y)
			x += widths[j]
			if j < ncol-1 {
				x += colGap[j]
			}
		}
		out.W = max(out.W, x)
		y += rd
	}
	// Centre the array on the maths axis.
	shift := -y/2 - axisHeight*size
	for i := range out.Items {
		out.Items[i].Y += shift
		for k := range out.Items[i].Points {
			out.Items[i].Points[k][1] += shift
		}
	}
	out.H = -shift
	out.D = y + shift

	var l, r rune
	switch a.env {
	case "pmatrix":
		l, r = '(', ')'
	case "bmatrix":
		l, r = '[', ']'
	case "Bmatrix":
		l, r = '{', '}'
	case "vmatrix":
		l, r = '|', '|'
	case "Vmatrix":
		l, r = '‖', '‖'
	case "cases", "dcases":
		l = '{'
	default:
		return out, clsOrd
	}
	// A little room between the entries and the delimiters.
	pad := 0.12 * size
	var inner hlist
	inner.space(pad)
	inner.add(out, 0)
	inner.space(pad)
	if r == 0 {
		inner.b.W -= pad
	}
	return wrapDelims(&inner.b, l, r, st.scale()), clsInner
}
