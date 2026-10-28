package tex

// Stretchable delimiters built from the Unicode bracket pieces:
// top, extender, bottom and middle.
var delimPieces = map[rune][4]rune{
	'(': {'⎛', '⎜', '⎝', 0}, ')': {'⎞', '⎟', '⎠', 0},
	'[': {'⎡', '⎢', '⎣', 0}, ']': {'⎤', '⎥', '⎦', 0},
	'{': {'⎧', '⎪', '⎩', '⎨'}, '}': {'⎫', '⎪', '⎭', '⎬'},
	'⌈': {'⎡', '⎢', '⎢', 0}, '⌉': {'⎤', '⎥', '⎥', 0},
	'⌊': {'⎢', '⎢', '⎣', 0}, '⌋': {'⎥', '⎥', '⎦', 0},
}

// glyphRef is a glyph with its metrics.
type glyphRef struct {
	face uint8
	g    uint16
	m    glyphMetrics
}

func glyphOf(r rune) (glyphRef, bool) {
	face, g, ok := lookup(r)
	if !ok {
		return glyphRef{}, false
	}
	return glyphRef{face, g, glyphInfo(face, g)}, true
}

// height is the ink height of the glyph at em size size.
func (gr glyphRef) height(size float32) float32 { return (gr.m.maxY - gr.m.minY) * size }

// delimiter returns delimiter r with the given total height, centred on the
// maths axis. r == 0 is the null delimiter.
func delimiter(r rune, total, size float32) *Box {
	if r == 0 {
		return &Box{W: nullDelim * size}
	}
	gr, ok := glyphOf(r)
	if !ok {
		return glyphBox(r, size, false)
	}
	nat := gr.height(size)
	if total <= nat*1.05 {
		// The normal glyph is big enough.
		return axisGlyph(gr.face, gr.g, gr.m, size, axisHeight*size)
	}
	d := newDelimBuilder(gr, total, size)
	pieces, ok := delimPieces[r]
	if !ok || total < nat*1.4 {
		d.stretch(gr)
	} else {
		d.assemble(pieces)
	}
	return d.out
}

// delimBuilder builds a delimiter spanning top to bot (y down).
type delimBuilder struct {
	out      *Box
	size     float32
	top, bot float32
}

func newDelimBuilder(gr glyphRef, total, size float32) *delimBuilder {
	axis := axisHeight * size
	top, bot := -(total/2 + axis), total/2-axis
	return &delimBuilder{out: &Box{W: gr.m.adv * size, H: -top, D: bot}, size: size, top: top, bot: bot}
}

func (d *delimBuilder) add(gr glyphRef, y, ys float32) {
	d.out.Items = append(d.out.Items, glyphItem(gr.face, gr.g, d.size, ys, y))
}

// stretch scales the glyph vertically to the full height.
func (d *delimBuilder) stretch(gr glyphRef) {
	ys := (d.bot - d.top) / gr.height(d.size)
	d.add(gr, d.top-gr.m.minY*d.size*ys, ys)
}

// assemble builds the delimiter from its end pieces with extenders between
// them (and a middle piece for braces).
func (d *delimBuilder) assemble(pieces [4]rune) {
	top, _ := glyphOf(pieces[0])
	ext, _ := glyphOf(pieces[1])
	bottom, _ := glyphOf(pieces[2])
	d.out.W = max(top.m.adv, bottom.m.adv, ext.m.adv) * d.size
	d.add(top, d.top-top.m.minY*d.size, 1)
	d.add(bottom, d.bot-bottom.m.maxY*d.size, 1)
	topEnd := d.top + top.height(d.size)
	bottomStart := d.bot - bottom.height(d.size)
	if pieces[3] == 0 {
		d.extend(ext, topEnd, bottomStart)
		return
	}
	mid, _ := glyphOf(pieces[3])
	my := -axisHeight*d.size - (mid.m.minY+mid.m.maxY)/2*d.size
	d.add(mid, my, 1)
	d.extend(ext, topEnd, my+mid.m.minY*d.size)
	d.extend(ext, my+mid.m.maxY*d.size, bottomStart)
}

// extend stretches the extender glyph from y = from to y = to, overlapping
// the neighbouring pieces slightly.
func (d *delimBuilder) extend(ext glyphRef, from, to float32) {
	overlap := 0.02 * d.size
	span := to - from + 2*overlap
	eh := ext.height(d.size)
	if span <= 0 || eh <= 0 {
		return
	}
	ys := span / eh
	d.add(ext, from-overlap-ext.m.minY*d.size*ys, ys)
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

func leftRight(n *nLeftRight, st style) *Box {
	size := st.scale()
	parts := make([]*Box, len(n.parts))
	var all Box // the height and depth of all parts together
	for i, p := range n.parts {
		parts[i] = list(p, st)
		all.H = max(all.H, parts[i].H)
		all.D = max(all.D, parts[i].D)
	}
	total := delimHeight(&all, size)
	var h hlist
	for i, d := range n.delims {
		h.add(delimiter(d, total, size), 0)
		if i < len(parts) {
			h.add(parts[i], 0)
		}
	}
	return &h.b
}
