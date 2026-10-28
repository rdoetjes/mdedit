// Command glyphdump shows how a character becomes an outline in the Go font
// mdedit draws text with: its glyph number (cmap), where its data is (loca),
// the points as stored (glyf), and the curve segments built from them.
//
//	go run ./tools/glyphdump o
//
// See HOW_IT_WORKS.md, "From a character to pixels".
package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/font/opentype/tables"
	"golang.org/x/image/font/gofont/goregular"
)

func main() {
	ch := 'o'
	if len(os.Args) > 1 {
		ch = []rune(os.Args[1])[0]
	}
	face, err := font.ParseTTF(bytes.NewReader(goregular.TTF))
	check(err)
	gid, ok := face.NominalGlyph(ch)
	if !ok {
		fmt.Printf("%q is not in the Go font (glyph 0, the missing-glyph box)\n", ch)
		return
	}
	fmt.Printf("char %q U+%04X -> glyph %d, %d units per em\n\n", ch, ch, gid, face.Upem())
	dumpPoints(gid)
	dumpSegments(face, gid)
}

// glyfTable reads the loca offsets and the parsed glyf table.
func glyfTable() ([]uint32, tables.Glyf) {
	ld, err := ot.NewLoader(bytes.NewReader(goregular.TTF))
	check(err)
	head, _, err := tables.ParseHead(table(ld, "head"))
	check(err)
	maxp, _, err := tables.ParseMaxp(table(ld, "maxp"))
	check(err)
	loca, err := tables.ParseLoca(table(ld, "loca"), int(maxp.NumGlyphs), head.IndexToLocFormat == 1)
	check(err)
	glyf, err := tables.ParseGlyf(table(ld, "glyf"), loca)
	check(err)
	return loca, glyf
}

func table(ld *ot.Loader, tag string) []byte {
	b, err := ld.RawTable(ot.MustNewTag(tag))
	check(err)
	return b
}

// dumpPoints prints the glyph's points as stored in the glyf table.
func dumpPoints(gid font.GID) {
	loca, glyf := glyfTable()
	fmt.Printf("loca: glyph %d is bytes %d..%d of the glyf table\n", gid, loca[gid], loca[gid+1])
	sg, simple := glyf[gid].Data.(tables.SimpleGlyph)
	if !simple {
		fmt.Printf("composite glyph: built from other glyphs %v\n\n", glyf[gid].Data)
		return
	}
	fmt.Printf("contours end at points %v\n", sg.EndPtsOfContours)
	for i, p := range sg.Points {
		fmt.Printf("  point %2d  flag 0x%02x  (%5d, %5d)  %s\n", i, p.Flag, p.X, p.Y, pointKind(p.Flag))
	}
	fmt.Println()
}

func pointKind(flag uint8) string {
	if flag&1 != 0 {
		return "on-curve"
	}
	return "off-curve (control)"
}

var segNames = map[ot.SegmentOp]string{
	ot.SegmentOpMoveTo: "MoveTo", ot.SegmentOpLineTo: "LineTo",
	ot.SegmentOpQuadTo: "QuadTo", ot.SegmentOpCubeTo: "CubeTo",
}

var segArgs = map[ot.SegmentOp]int{
	ot.SegmentOpMoveTo: 1, ot.SegmentOpLineTo: 1, ot.SegmentOpQuadTo: 2, ot.SegmentOpCubeTo: 3,
}

// dumpSegments prints the curve segments the font library builds.
func dumpSegments(face *font.Face, gid font.GID) {
	out, ok := face.GlyphData(gid).(font.GlyphOutline)
	if !ok {
		fmt.Println("not an outline glyph")
		return
	}
	fmt.Println("segments (font units, y up):")
	for _, s := range out.Segments {
		fmt.Printf("  %-6s", segNames[s.Op])
		for _, a := range s.Args[:segArgs[s.Op]] {
			fmt.Printf(" (%6.1f, %6.1f)", a.X, a.Y)
		}
		fmt.Println()
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "glyphdump:", err)
		os.Exit(1)
	}
}
