// Command mkicon draws the MDEdit application icon as PNG files in the
// standard icon theme sizes: a blue-violet tile with a white card showing
// the Markdown mark (M↓) and a small sigma for the maths support.
//
// Usage:
//
//	go run ./tools/mkicon <output dir>            hicolor theme PNGs (Linux, BSD)
//	go run ./tools/mkicon -macos <name.iconset>   macOS iconset for iconutil
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/image/vector"
)

// sizes are the hicolor icon theme sizes to generate.
var sizes = []int{16, 24, 32, 48, 64, 128, 256, 512}

// The design is drawn on a 128×128 grid.
const grid = 128

var (
	gradTop    = color.NRGBA{R: 0x3b, G: 0x82, B: 0xf6, A: 0xff}
	gradBottom = color.NRGBA{R: 0x7c, G: 0x3a, B: 0xed, A: 0xff}
	cardShadow = color.NRGBA{R: 0x1e, G: 0x1b, B: 0x4b, A: 0x55}
	cardWhite  = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	markBlue   = color.NRGBA{R: 0x25, G: 0x3e, B: 0xa8, A: 0xff}
)

var macos = flag.Bool("macos", false, "write a macOS .iconset (for iconutil) instead of hicolor PNGs")

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: mkicon [-macos] <output dir>")
		os.Exit(2)
	}
	write := writeHicolor
	if *macos {
		write = writeIconset
	}
	if err := write(flag.Arg(0)); err != nil {
		fmt.Fprintln(os.Stderr, "mkicon:", err)
		os.Exit(1)
	}
}

// writeHicolor saves <dir>/<s>x<s>/apps/mdedit.png for each theme size.
func writeHicolor(dir string) error {
	for _, s := range sizes {
		path := filepath.Join(dir, fmt.Sprintf("%dx%d", s, s), "apps", "mdedit.png")
		if err := savePNG(path, drawIcon(s, fullTile(s))); err != nil {
			return err
		}
	}
	return nil
}

// iconsetSizes are the point sizes of a macOS iconset; each comes at 1x and
// 2x.
var iconsetSizes = []int{16, 32, 128, 256, 512}

// writeIconset saves a macOS iconset: icon_<s>x<s>[@2x].png.
func writeIconset(dir string) error {
	for _, s := range iconsetSizes {
		for _, scale := range []int{1, 2} {
			name := fmt.Sprintf("icon_%dx%d.png", s, s)
			if scale == 2 {
				name = fmt.Sprintf("icon_%dx%d@2x.png", s, s)
			}
			px := s * scale
			if err := savePNG(filepath.Join(dir, name), drawIcon(px, macTile(px))); err != nil {
				return err
			}
		}
	}
	return nil
}

func savePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// placement maps the 128 grid onto an icon of size px: pixel = off + v*scale.
type placement struct {
	scale, off float32
	shadow     bool
}

// fullTile fills the icon with the tile, as icon themes expect.
func fullTile(px int) placement { return placement{scale: float32(px) / grid} }

// macTile follows Apple's icon grid: the 112-unit tile becomes 824 of 1024
// pixels, centred, leaving room for a soft shadow.
func macTile(px int) placement {
	scale := float32(px) * 824 / 1024 / 112
	return placement{scale: scale, off: float32(px)/2 - 64*scale, shadow: true}
}

var tileShadow = color.NRGBA{R: 0x10, G: 0x10, B: 0x30, A: 0x30}

func drawIcon(s int, p placement) *image.NRGBA {
	c := &canvas{img: image.NewNRGBA(image.Rect(0, 0, s, s)), scale: p.scale, off: p.off}
	if p.shadow {
		c.fill(roundRect(8, 10, 120, 123, 26), image.NewUniform(tileShadow))
		c.fill(roundRect(8, 9, 120, 121, 26), image.NewUniform(tileShadow))
	}
	c.fill(roundRect(8, 8, 120, 120, 26), gradient{size: s})
	c.fill(roundRect(22, 41, 106, 95, 10), image.NewUniform(cardShadow))
	c.fill(roundRect(22, 37, 106, 91, 10), image.NewUniform(cardWhite))
	c.fill(letterM(), image.NewUniform(markBlue))
	c.fill(downArrow(), image.NewUniform(markBlue))
	c.fill(sigma(), image.NewUniform(cardWhite))
	return c.img
}

// shape is a list of closed polygons, on the 128 grid. Curves are
// flattened into points beforehand.
type shape [][][2]float32

// canvas rasterises shapes onto an image.
type canvas struct {
	img   *image.NRGBA
	scale float32
	off   float32
}

// fill paints the shape with src, anti-aliased.
func (c *canvas) fill(sh shape, src image.Image) {
	b := c.img.Bounds()
	r := vector.NewRasterizer(b.Dx(), b.Dy())
	for _, poly := range sh {
		c.addPolygon(r, poly)
	}
	r.Draw(c.img, b, src, image.Point{})
}

func (c *canvas) addPolygon(r *vector.Rasterizer, poly [][2]float32) {
	r.MoveTo(c.off+poly[0][0]*c.scale, c.off+poly[0][1]*c.scale)
	for _, p := range poly[1:] {
		r.LineTo(c.off+p[0]*c.scale, c.off+p[1]*c.scale)
	}
	r.ClosePath()
}

// gradient runs diagonally from gradTop to gradBottom.
type gradient struct{ size int }

func (g gradient) ColorModel() color.Model { return color.NRGBAModel }
func (g gradient) Bounds() image.Rectangle { return image.Rect(0, 0, g.size, g.size) }

func (g gradient) At(x, y int) color.Color {
	t := float32(x+y) / float32(2*g.size)
	return color.NRGBA{
		R: mix(gradTop.R, gradBottom.R, t),
		G: mix(gradTop.G, gradBottom.G, t),
		B: mix(gradTop.B, gradBottom.B, t),
		A: 0xff,
	}
}

func mix(a, b uint8, t float32) uint8 {
	return uint8(float32(a)*(1-t) + float32(b)*t + 0.5)
}

// roundRect is a rectangle with circular corners of radius r.
func roundRect(x0, y0, x1, y1, r float32) shape {
	var pts [][2]float32
	pts = append(pts, arc(x1-r, y0+r, r, -90, 0)...)
	pts = append(pts, arc(x1-r, y1-r, r, 0, 90)...)
	pts = append(pts, arc(x0+r, y1-r, r, 90, 180)...)
	pts = append(pts, arc(x0+r, y0+r, r, 180, 270)...)
	return shape{pts}
}

// arc returns points on a circle from angle a0 to a1 (degrees, y down).
func arc(cx, cy, r, a0, a1 float32) [][2]float32 {
	const steps = 12
	out := make([][2]float32, 0, steps+1)
	for i := 0; i <= steps; i++ {
		a := float64(a0+(a1-a0)*float32(i)/steps) * math.Pi / 180
		out = append(out, [2]float32{cx + r*float32(math.Cos(a)), cy + r*float32(math.Sin(a))})
	}
	return out
}

// letterM is a bold M on the card.
func letterM() shape {
	return shape{{
		{33, 81}, {33, 48}, {43, 48}, {52, 61}, {61, 48}, {71, 48}, {71, 81},
		{62, 81}, {62, 63}, {52, 76}, {42, 63}, {42, 81},
	}}
}

// downArrow is the arrow of the Markdown mark.
func downArrow() shape {
	return shape{{
		{80, 48}, {89, 48}, {89, 64}, {96, 64}, {84.5, 80}, {73, 64}, {80, 64},
	}}
}

// sigma is a small Σ centred below the card.
func sigma() shape {
	return shape{{
		{55, 99}, {73, 99}, {73, 103}, {61, 103}, {67, 107.5}, {61, 112},
		{73, 112}, {73, 116}, {55, 116}, {55, 113}, {62, 107.5}, {55, 102},
	}}
}
