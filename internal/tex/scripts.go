package tex

// scripts lays out an atom with superscript and/or subscript, either beside
// it or, for large operators in display style, above and below it.
func scripts(n *nScripts, st style) (*Box, class) {
	sb := scriptBase(n, st)
	sst := st.script()
	var sup, sub *Box
	if n.sup != nil {
		sup, _ = layoutNode(n.sup, sst)
	}
	if n.sub != nil {
		sub, _ = layoutNode(n.sub, sst)
	}
	if sb.limits {
		return limitsBox(sb.box, sup, sub, st.scale()), sb.cls
	}
	_, isChar := n.base.(*nSym)
	u, v := scriptShifts(sb.box, sup, sub, st, isChar)
	return attachScripts(sb, sup, sub, u, v, st.scale()), sb.cls
}

// base is the laid-out nucleus of a scripts node.
type base struct {
	box      *Box
	cls      class
	integral bool
	limits   bool // scripts go above and below
}

func scriptBase(n *nScripts, st style) base {
	b := base{limits: n.limits == 1}
	switch nb := n.base.(type) {
	case *nOp:
		b.box, b.integral = opBox(nb, st)
		b.cls = clsOp
		if nb.limits && st == sD && n.limits != -1 {
			b.limits = true
		}
	case *nAccent:
		if nb.kind == "overbrace" || nb.kind == "underbrace" {
			b.limits = n.limits != -1
		}
		b.box, b.cls = layoutNode(nb, st)
	default:
		b.box, b.cls = layoutNode(n.base, st)
	}
	return b
}

// limitsBox stacks the scripts centred above and below the base.
func limitsBox(b, sup, sub *Box, size float32) *Box {
	w := b.W
	if sup != nil {
		w = max(w, sup.W)
	}
	if sub != nil {
		w = max(w, sub.W)
	}
	out := &Box{W: w, H: b.H, D: b.D}
	out.place(b, (w-b.W)/2, 0)
	gap := 0.12 * size
	if sup != nil {
		out.place(sup, (w-sup.W)/2, -(b.H + gap + sup.D))
		out.H = b.H + gap + sup.D + sup.H
	}
	if sub != nil {
		out.place(sub, (w-sub.W)/2, b.D+gap+sub.H)
		out.D = b.D + gap + sub.H + sub.D
	}
	return out
}

// scriptShifts returns how far the superscript is raised (u) and the
// subscript lowered (v), following TeX's rules 18a–f.
func scriptShifts(b, sup, sub *Box, st style, isChar bool) (u, v float32) {
	size, ss := st.scale(), st.script().scale()
	if !isChar {
		u = b.H - 0.25*ss
		v = b.D + 0.05*ss
	}
	if sup != nil {
		u = max(u, minSupShift(st)*size, sup.D+xHeight*size/4)
	}
	if sub != nil {
		v = max(v, 0.15*size, sub.H-0.8*xHeight*size)
	}
	if sup != nil && sub != nil {
		u, v = separateScripts(u, v, sup, sub, size)
	}
	return u, v
}

func minSupShift(st style) float32 {
	if st == sD {
		return 0.413
	}
	return 0.363
}

// separateScripts keeps a gap between a superscript and a subscript.
func separateScripts(u, v float32, sup, sub *Box, size float32) (float32, float32) {
	v = max(v, 0.247*size)
	gap := (u - sup.D) - (sub.H - v)
	if gap >= 4*ruleWidth*size {
		return u, v
	}
	v += 4*ruleWidth*size - gap
	if psi := 0.8*xHeight*size - (u - sup.D); psi > 0 {
		u += psi
		v -= psi
	}
	return u, v
}

// attachScripts places the scripts beside the base; an integral's subscript
// tucks under its slant.
func attachScripts(b base, sup, sub *Box, u, v, size float32) *Box {
	supX, subX := b.box.W, b.box.W
	if b.integral {
		subX = b.box.W * 0.65
		supX = b.box.W * 1.02
	}
	out := &Box{W: b.box.W, H: b.box.H, D: b.box.D}
	out.place(b.box, 0, 0)
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
	return out
}
