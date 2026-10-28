package render

import (
	"testing"

	"mdedit/internal/mdmodel"
)

func testCtx(controls bool, w, h int) *Ctx {
	return &Ctx{Em: 16, Theme: &Light, Measure: fixedMeasure, VideoControls: controls,
		ImageSize: imageOf{w, h}.size}
}

type imageOf struct{ w, h int }

func (i imageOf) size(string) (int, int, bool) { return i.w, i.h, i.w > 0 }

func fixedMeasure(_ FontSpec, size float32, s string) Metrics {
	return Metrics{Width: float32(len(s)) * size / 2, Ascent: size * 0.9, Descent: size * 0.2}
}

func video() *mdmodel.Block {
	return &mdmodel.Block{Kind: mdmodel.Image, Src: "https://img.youtube.com/vi/dQw4w9WgXcQ/maxresdefault.jpg",
		Link: "https://www.youtube.com/watch?v=dQw4w9WgXcQ", Video: true}
}

func countKind(l *Layout, k PrimKind) int {
	n := 0
	for _, p := range l.Prims {
		if p.Kind == k {
			n++
		}
	}
	return n
}

func imagePrim(t *testing.T, l *Layout) Prim {
	for _, p := range l.Prims {
		if p.Kind == PImage {
			return p
		}
	}
	t.Fatal("no image")
	return Prim{}
}

// A video still is 66% of the width, whatever its pixel size, and links
// to the video.
func TestVideoSize(t *testing.T) {
	for _, px := range [][2]int{{1280, 720}, {480, 360}} {
		l := testCtx(true, px[0], px[1]).Block(video(), nil, 1000)
		img := imagePrim(t, l)
		if img.W != 660 || img.H != 660*float32(px[1])/float32(px[0]) {
			t.Errorf("%v: still %vx%v, want 660 wide", px, img.W, img.H)
		}
		if countKind(l, PLink) != 1 || countKind(l, PPoly) != 1 {
			t.Errorf("%v: want a link and a play button", px)
		}
	}
}

// The PDF shows just the image: no play button.
func TestVideoInPDF(t *testing.T) {
	l := testCtx(false, 1280, 720).Block(video(), nil, 1000)
	if countKind(l, PPoly) != 0 || countKind(l, PImage) != 1 {
		t.Error("PDF layout should have the image and no play button")
	}
}

// While the still downloads, the preview keeps a 16:9 placeholder.
func TestVideoLoading(t *testing.T) {
	l := testCtx(true, 0, 0).Block(video(), nil, 1000)
	if l.Height != 660*9.0/16 || countKind(l, PImage) != 0 {
		t.Errorf("placeholder height %v, want %v", l.Height, 660*9.0/16)
	}
}
