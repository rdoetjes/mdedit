package tex

// arrayLayout lays out matrices, cases and aligned equations.
type arrayLayout struct {
	a       *nArray
	cst     style // style of the cells
	size    float32
	ncol    int
	align   []byte    // 'l', 'c' or 'r' per column
	colGap  []float32 // space after each column
	aligned bool      // aligned equations: r, l, r, l … columns
	cells   [][]*Box
	widths  []float32
}

func array(a *nArray, st style) (*Box, class) {
	l := &arrayLayout{a: a, cst: cellStyle(a.env, st)}
	l.size = l.cst.scale()
	for _, r := range a.rows {
		l.ncol = max(l.ncol, len(r))
	}
	if l.ncol == 0 {
		return &Box{}, clsOrd
	}
	l.columns()
	l.layoutCells()
	return l.withDelimiters(l.stack(), st)
}

// cellStyle is display style for display equations, script style for small
// matrices, and text style otherwise.
func cellStyle(env string, st style) style {
	cst := sT
	switch env {
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
	return cst
}

// columns sets the alignment of and the gaps between the columns.
func (l *arrayLayout) columns() {
	l.align = make([]byte, l.ncol)
	l.colGap = make([]float32, l.ncol)
	for i := range l.align {
		l.align[i], l.colGap[i] = 'c', 1.0*l.size
	}
	switch l.a.env {
	case "aligned", "align", "alignat", "split":
		l.alignedColumns()
	case "cases", "dcases":
		for i := range l.align {
			l.align[i] = 'l'
		}
	case "array":
		l.arrayColumns()
	case "smallmatrix":
		for i := range l.colGap {
			l.colGap[i] = 0.5 * l.size
		}
	}
}

// alignedColumns pairs right- and left-aligned columns, with no gap within
// a pair: "a &= b".
func (l *arrayLayout) alignedColumns() {
	l.aligned = true
	for i := range l.align {
		if i%2 == 0 {
			l.align[i], l.colGap[i] = 'r', 0
		} else {
			l.align[i], l.colGap[i] = 'l', 1.5*l.size
		}
	}
}

// arrayColumns takes the alignments from the column spec, e.g. {lcr}.
func (l *arrayLayout) arrayColumns() {
	j := 0
	for _, c := range l.a.cols {
		if (c == 'l' || c == 'c' || c == 'r') && j < l.ncol {
			l.align[j] = byte(c)
			j++
		}
	}
}

func (l *arrayLayout) layoutCells() {
	l.cells = make([][]*Box, len(l.a.rows))
	l.widths = make([]float32, l.ncol)
	for i, r := range l.a.rows {
		l.cells[i] = make([]*Box, l.ncol)
		for j := range l.cells[i] {
			b := list(l.cellNodes(r, j), l.cst)
			l.cells[i][j] = b
			l.widths[j] = max(l.widths[j], b.W)
		}
	}
}

// cellNodes returns the nodes of cell j; in aligned equations a leading
// relation gets its left spacing: "&= b".
func (l *arrayLayout) cellNodes(r [][]node, j int) []node {
	var nodes []node
	if j < len(r) {
		nodes = r[j]
	}
	if l.aligned && j%2 == 1 {
		nodes = append([]node{&nGroup{}}, nodes...)
	}
	return nodes
}

// stack places the rows below each other and centres them on the axis.
func (l *arrayLayout) stack() *Box {
	rowGap := 0.25 * l.size
	if l.cst == sD {
		rowGap = 0.4 * l.size
	}
	out := &Box{}
	y := float32(0)
	for i := range l.cells {
		if i > 0 {
			y += rowGap
		}
		y = l.row(out, i, y)
	}
	return centerOnAxis(out, y, l.size)
}

// row places row i with its top at y and returns the y below it.
func (l *arrayLayout) row(out *Box, i int, y float32) float32 {
	rh, rd := 0.7*l.size, 0.3*l.size
	for _, b := range l.cells[i] {
		rh, rd = max(rh, b.H), max(rd, b.D)
	}
	y += rh
	x := float32(0)
	for j, b := range l.cells[i] {
		out.place(b, x+l.alignOffset(j, b), y)
		x += l.widths[j]
		if j < l.ncol-1 {
			x += l.colGap[j]
		}
	}
	out.W = max(out.W, x)
	return y + rd
}

func (l *arrayLayout) alignOffset(j int, b *Box) float32 {
	switch l.align[j] {
	case 'c':
		return (l.widths[j] - b.W) / 2
	case 'r':
		return l.widths[j] - b.W
	}
	return 0
}

// centerOnAxis moves a box of the given total height so its middle is on
// the maths axis.
func centerOnAxis(b *Box, height, size float32) *Box {
	shift := -height/2 - axisHeight*size
	out := &Box{W: b.W, H: -shift, D: height + shift}
	out.place(b, 0, shift)
	return out
}

var envDelims = map[string][2]rune{
	"pmatrix": {'(', ')'}, "bmatrix": {'[', ']'}, "Bmatrix": {'{', '}'},
	"vmatrix": {'|', '|'}, "Vmatrix": {'‖', '‖'}, "cases": {'{', 0}, "dcases": {'{', 0},
}

// withDelimiters adds the environment's delimiters, with a little room
// between them and the entries.
func (l *arrayLayout) withDelimiters(out *Box, st style) (*Box, class) {
	d, ok := envDelims[l.a.env]
	if !ok {
		return out, clsOrd
	}
	pad := 0.12 * l.size
	var inner hlist
	inner.space(pad)
	inner.add(out, 0)
	if d[1] != 0 {
		inner.space(pad)
	}
	return wrapDelims(&inner.b, d[0], d[1], st.scale()), clsInner
}
