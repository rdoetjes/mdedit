package render

import (
	"image/color"

	"mdedit/internal/mdmodel"
)

// videoShare is the largest share of the width a video still takes.
const videoShare = 0.66

var (
	videoPlaceholder = color.NRGBA{R: 0x24, G: 0x29, B: 0x2f, A: 0xff}
	playRed          = color.NRGBA{R: 0xff, G: 0x00, B: 0x00, A: 0xe6}
	playWhite        = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
)

// image lays out an image block; a missing image is shown by its alt text.
func (c *Ctx) image(b *mdmodel.Block, w float32, col color.NRGBA) (*Layout, float32) {
	pw, ph, ok := c.imageSize(b.Src)
	loading := !ok && b.Video && c.VideoControls
	if loading {
		pw, ph, ok = 16, 9, true // keep the space of a 16:9 still
	}
	if !ok || pw == 0 || ph == 0 {
		return c.altText(b, w, col)
	}
	iw, ih := c.fitImage(b, float32(pw), float32(ph), w)
	x := alignedX(b.Align, w, iw)
	l := &Layout{Height: ih}
	if loading {
		l.Prims = append(l.Prims, Prim{Kind: PRect, X: x, W: iw, H: ih, Radius: 0.4 * c.Em, Color: videoPlaceholder})
	} else {
		l.Prims = append(l.Prims, Prim{Kind: PImage, X: x, W: iw, H: ih, URL: b.Src})
	}
	if b.Video && c.VideoControls {
		l.Prims = append(l.Prims, playButton(x, iw, ih)...)
	}
	if b.Link != "" {
		l.Prims = append(l.Prims, Prim{Kind: PLink, X: x, W: iw, H: ih, URL: b.Link})
	}
	return l, min(ih, c.Em)
}

func (c *Ctx) imageSize(src string) (int, int, bool) {
	if c.ImageSize == nil {
		return 0, 0, false
	}
	return c.ImageSize(src)
}

// altText stands in for an image that can't be shown.
func (c *Ctx) altText(b *mdmodel.Block, w float32, col color.NRGBA) (*Layout, float32) {
	alt, target := b.Alt, b.Link
	if alt == "" {
		alt = b.Src
	}
	if target == "" {
		target = b.Src
	}
	return c.paragraph([]mdmodel.Span{{Text: "[image: " + alt + "]", Italic: true, Link: target}}, w, col, b.Align)
}

// fitImage returns the displayed size of an image of pw×ph pixels. Video
// stills take videoShare of the width; other images use their own size
// (pixels as CSS pixels, 16 per em) but no more than the width.
func (c *Ctx) fitImage(b *mdmodel.Block, pw, ph, w float32) (float32, float32) {
	iw, ih := pw*c.Em/16, ph*c.Em/16
	if b.Video {
		iw, ih = w*videoShare, w*videoShare*ph/pw
	}
	if iw > w {
		iw, ih = w, ih*w/iw
	}
	if c.MaxImageHeight > 0 && ih > c.MaxImageHeight {
		iw, ih = iw*c.MaxImageHeight/ih, c.MaxImageHeight
	}
	return iw, ih
}

func alignedX(align mdmodel.Align, w, iw float32) float32 {
	switch align {
	case mdmodel.AlignCenter:
		return (w - iw) / 2
	case mdmodel.AlignRight:
		return w - iw
	}
	return 0
}

// playButton is a YouTube-style play button centred on a still at x.
func playButton(x, iw, ih float32) []Prim {
	bw := max(iw*0.14, 24)
	bh := bw * 0.7
	cx, cy := x+iw/2, ih/2
	t := bh * 0.22 // half the triangle's height
	return []Prim{
		{Kind: PRect, X: cx - bw/2, Y: cy - bh/2, W: bw, H: bh, Radius: bh * 0.28, Color: playRed},
		{Kind: PPoly, Color: playWhite, Points: [][2]float32{
			{cx - t*0.8, cy - t}, {cx + t*1.1, cy}, {cx - t*0.8, cy + t},
		}},
	}
}
