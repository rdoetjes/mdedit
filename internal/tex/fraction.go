package tex

// fraction lays out \frac, \dfrac, \tfrac and \binom.
func fraction(n *nFrac, st style) *Box {
	fst := fractionStyle(n, st)
	size := fst.scale()
	num, _ := layoutNode(n.num, fst.fraction())
	den, _ := layoutNode(n.den, fst.fraction())
	rule, pad := float32(0), float32(0)
	if n.bar {
		rule, pad = ruleWidth*size, nullDelim*size
	}
	numShift, denShift := fractionShifts(fst, n.bar, rule, num, den)
	w := max(num.W, den.W) + 2*pad
	out := &Box{W: w, H: numShift + num.H, D: denShift + den.D}
	out.place(num, (w-num.W)/2, -numShift)
	out.place(den, (w-den.W)/2, denShift)
	if n.bar {
		out.Items = append(out.Items, Item{Kind: ItemRule, X: pad / 2, Y: -axisHeight*size - rule/2, W: w - pad, H: rule})
	}
	if n.left != 0 || n.right != 0 {
		return wrapDelims(out, n.left, n.right, size)
	}
	return out
}

// fractionStyle is the style the fraction is set in: \dfrac forces display
// style and \tfrac at most text style.
func fractionStyle(n *nFrac, st style) style {
	switch n.style {
	case 0:
		return sD
	case 1:
		return max(st, sT)
	}
	return st
}

// fractionShifts returns how far the numerator is raised and the
// denominator lowered, keeping them clear of the bar.
func fractionShifts(fst style, bar bool, rule float32, num, den *Box) (float32, float32) {
	size := fst.scale()
	numShift, denShift, gap := 0.394*size, 0.345*size, ruleWidth*size
	if fst == sD {
		numShift, denShift, gap = 0.677*size, 0.686*size, 3*ruleWidth*size
	}
	if !bar {
		gap *= 2
	}
	axis := axisHeight * size
	numShift = max(numShift, axis+rule/2+gap+num.D)
	denShift = max(denShift, -axis+rule/2+gap+den.H)
	return numShift, denShift
}

// radical holds the geometry of a square root sign.
type radical struct {
	size     float32
	rule     float32
	innerTop float32 // top of the vinculum above the baseline
	bottom   float32 // depth of the sign below the baseline
	total    float32
	sw       float32 // width of the sign
	tickX    float32 // x of the top of the small tick
}

func newRadical(body *Box, st style) radical {
	size := st.scale()
	rule := ruleWidth * size
	phi := rule
	if st == sD {
		phi = xHeight * size
	}
	r := radical{size: size, rule: rule, tickX: 0.12 * size}
	r.innerTop = body.H + rule + phi/4 + rule
	r.bottom = max(body.D, 0.1*size)
	r.total = r.innerTop + r.bottom
	r.sw = min(0.5*size+0.1*r.total, 1.1*size)
	return r
}

// sign draws the radical sign: a short rising tick, a heavy down-stroke and
// a long thin up-stroke meeting the vinculum.
func (r radical) sign() *Box {
	hs := min(r.total, 1.1*r.size)
	yTop := -r.innerTop + r.rule/2
	ay := r.bottom - 0.48*hs
	by := ay - 0.07*r.size
	cx := 0.28*r.size + 0.08*(r.sw-0.6*r.size)
	return &Box{W: r.sw, H: r.innerTop, D: r.bottom, Items: []Item{
		quad(0, ay, r.tickX, by, 0.045*r.size),
		quad(r.tickX, by, cx, r.bottom, 0.095*r.size),
		quad(cx, r.bottom, r.sw, yTop, 0.045*r.size),
	}}
}

// index places a root index over the tick, returning it as a zero-width box
// and how far the sign must move right to make room for it.
func (r radical) index(idx *Box) (*Box, float32) {
	room := r.tickX + 0.06*r.size
	shift := max(0, idx.W-room)
	var ib Box
	ib.place(idx, shift+room-idx.W, r.bottom-0.6*r.total-idx.D)
	ib.H = 0.6*r.total - r.bottom + idx.D + idx.H
	return &ib, shift
}

// vinculum is the bar over the body, as a zero-width box.
func (r radical) vinculum(bodyW float32) *Box {
	return &Box{H: r.innerTop, Items: []Item{{Kind: ItemRule, X: -0.02 * r.size, Y: -r.innerTop,
		W: bodyW + 0.1*r.size + 0.02*r.size, H: r.rule}}}
}

func sqrt(n *nSqrt, st style) *Box {
	body, _ := layoutNode(n.body, st)
	r := newRadical(body, st)
	var h hlist
	if n.index != nil {
		idx, _ := layoutNode(n.index, sSS)
		ib, shift := r.index(idx)
		h.add(ib, 0)
		h.space(shift)
	}
	h.add(r.sign(), 0)
	h.add(r.vinculum(body.W), 0)
	h.add(body, 0)
	h.space(0.1 * r.size)
	h.b.H = max(h.b.H, r.innerTop+r.rule)
	return &h.b
}
