package tex

import "strings"

// accented lays out a body with something above, below or over it.
type accented struct {
	kind   string
	size   float32
	rule   float32
	body   *Box
	cls    class
	single bool    // the body is a single symbol
	skew   float32 // shift of accents over italic letters
}

func accent(n *nAccent, st style) (*Box, class) {
	size := st.scale()
	body, cls := layoutNode(n.body, st)
	a := &accented{kind: n.kind, size: size, rule: ruleWidth * size, body: body, cls: cls}
	if s, ok := n.body.(*nSym); ok {
		a.single = true
		if s.font == fItalic && isLetter(s.r) {
			a.skew = 0.08 * size
		}
	}
	return a.layout()
}

func (a *accented) layout() (*Box, class) {
	switch a.kind {
	case "overline", "bar":
		return a.overline(), a.cls
	case "underline":
		return a.underline(), a.cls
	case "vec", "overrightarrow", "overleftarrow":
		return a.arrow(), a.cls
	case "overbrace", "underbrace":
		return a.brace(), clsOp
	case "boxed":
		return a.boxed(), a.cls
	case "cancel", "not":
		return a.struckOut(), a.cls
	case "phantom", "hphantom", "vphantom":
		return a.phantom(), a.cls
	}
	return a.glyphAccent(), a.cls
}

// withBody returns a box containing the body, to add the accent to.
func (a *accented) withBody() *Box {
	out := &Box{W: a.body.W, H: a.body.H, D: a.body.D}
	out.place(a.body, 0, 0)
	return out
}

// overline draws \overline over the body, or a shorter \bar over a symbol.
func (a *accented) overline() *Box {
	out := a.withBody()
	w, x := a.body.W, float32(0)
	if a.kind == "bar" && a.single {
		w = a.body.W * 0.8
		x = (a.body.W-w)/2 + a.skew
	}
	y := -(a.body.H + 4*a.rule)
	out.Items = append(out.Items, Item{Kind: ItemRule, X: x, Y: y, W: w, H: a.rule})
	out.H = -y + a.rule
	return out
}

func (a *accented) underline() *Box {
	out := a.withBody()
	out.Items = append(out.Items, Item{Kind: ItemRule, Y: a.body.D + 3*a.rule, W: a.body.W, H: a.rule})
	out.D = a.body.D + 5*a.rule
	return out
}

// arrow draws \vec (a short arrow) or an arrow across the whole body.
func (a *accented) arrow() *Box {
	out := a.withBody()
	w, x := a.body.W, float32(0)
	if a.kind == "vec" {
		w = max(0.45*a.size, a.body.W*0.7)
		x = (a.body.W-w)/2 + a.skew
	}
	y := -(a.body.H + 0.15*a.size)
	th, head := 0.045*a.size, 0.13*a.size
	tip, back := x+w, x+w-head
	if a.kind == "overleftarrow" {
		tip, back = x, x+head
	}
	out.Items = append(out.Items, quad(x, y, x+w, y, th),
		quad(tip, y, back, y-head*0.7, th), quad(tip, y, back, y+head*0.7, th))
	out.H = -y + head*0.8
	return out
}

// brace draws a horizontal brace above or below the body.
func (a *accented) brace() *Box {
	out := a.withBody()
	th, hgt := 0.05*a.size, 0.2*a.size
	var y0, y1 float32 // the ends of the brace and its tip
	if a.kind == "overbrace" {
		y0 = -(a.body.H + 0.1*a.size)
		y1 = y0 - hgt
		out.H = -y1
	} else {
		y0 = a.body.D + 0.1*a.size
		y1 = y0 + hgt
		out.D = y1
	}
	out.Items = append(out.Items, braceStrokes(a.body.W, y0, y1, hgt, th)...)
	return out
}

func braceStrokes(w, y0, y1, hgt, th float32) []Item {
	ym := (y0 + y1) / 2
	return []Item{
		quad(0, y0, hgt/2, ym, th), quad(hgt/2, ym, w/2-hgt/2, ym, th),
		quad(w/2-hgt/2, ym, w/2, y1, th), quad(w/2, y1, w/2+hgt/2, ym, th),
		quad(w/2+hgt/2, ym, w-hgt/2, ym, th), quad(w-hgt/2, ym, w, y0, th),
	}
}

// boxed draws a frame around the body.
func (a *accented) boxed() *Box {
	pad, t := 0.2*a.size, 0.04*a.size
	var b Box
	b.place(a.body, pad, 0)
	b.W, b.H, b.D = a.body.W+2*pad, a.body.H+pad, a.body.D+pad
	b.Items = append(b.Items,
		Item{Kind: ItemRule, X: 0, Y: -b.H, W: b.W, H: t},
		Item{Kind: ItemRule, X: 0, Y: b.D - t, W: b.W, H: t},
		Item{Kind: ItemRule, X: 0, Y: -b.H, W: t, H: b.H + b.D},
		Item{Kind: ItemRule, X: b.W - t, Y: -b.H, W: t, H: b.H + b.D})
	return &b
}

// struckOut draws \cancel (corner to corner) or \not (a short slash).
func (a *accented) struckOut() *Box {
	out := a.withBody()
	b := a.body
	if a.kind == "cancel" {
		out.Items = append(out.Items, quad(0, b.D, b.W, -b.H, 0.04*a.size))
	} else {
		out.Items = append(out.Items, quad(b.W*0.25, b.D+0.05*a.size, b.W*0.75, -b.H-0.05*a.size, 0.05*a.size))
	}
	return out
}

// phantom takes up the body's space (or only its width or height) without
// drawing it.
func (a *accented) phantom() *Box {
	switch a.kind {
	case "hphantom":
		return &Box{W: a.body.W}
	case "vphantom":
		return &Box{H: a.body.H, D: a.body.D}
	}
	return &Box{W: a.body.W, H: a.body.H, D: a.body.D}
}

// glyphAccent puts an accent character (\hat, \tilde …) over the body,
// stretching it for the wide variants.
func (a *accented) glyphAccent() *Box {
	out := a.withBody()
	gr, ok := glyphOf(accents[a.kind])
	if !ok {
		return out
	}
	m := gr.m
	inkW := (m.maxX - m.minX) * a.size
	xs := float32(1)
	if strings.HasPrefix(a.kind, "wide") && inkW > 0 {
		xs = max(1, a.body.W*0.9/inkW)
	}
	gap := 0.05 * a.size
	it := glyphItem(gr.face, gr.g, a.size, 1, -(a.body.H+gap)-m.maxY*a.size)
	it.XScale = xs
	it.X = (a.body.W-inkW*xs)/2 - m.minX*a.size*xs + a.skew
	out.Items = append(out.Items, it)
	out.H = max(out.H, a.body.H+gap+gr.height(a.size))
	return out
}
