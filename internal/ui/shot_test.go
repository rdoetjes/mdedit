package ui

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// TestScreenshot renders the window offscreen. Set MDEDIT_SHOT to a directory
// to run it; useful for checking the UI without a display.
func TestScreenshot(t *testing.T) {
	dir := os.Getenv("MDEDIT_SHOT")
	if dir == "" {
		t.Skip("MDEDIT_SHOT not set")
	}
	a, err := New("../mdmodel/testdata/TEST.md")
	if err != nil {
		t.Fatal(err)
	}
	size := image.Pt(2560, 1640)
	w, err := headless.NewWindow(size.X, size.Y)
	if err != nil {
		t.Fatal(err)
	}
	var router input.Router
	shot := func(name string) {
		var ops op.Ops
		// Lay out a few times so scrolling and focus settle.
		for i := 0; i < 3; i++ {
			ops.Reset()
			gtx := layout.Context{
				Ops:         &ops,
				Now:         time.Now(),
				Metric:      unit.Metric{PxPerDp: 2, PxPerSp: 2},
				Constraints: layout.Exact(size),
				Source:      router.Source(),
			}
			a.layout(gtx)
			router.Frame(&ops)
		}
		if err := w.Frame(&ops); err != nil {
			t.Fatal(err)
		}
		img := image.NewRGBA(image.Rectangle{Max: size})
		if err := w.Screenshot(img); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		png.Encode(f, img)
	}
	shot("split.png")
	for _, id := range []string{"blockquotes", "lists", "code-blocks"} {
		a.preview.ScrollToBlock(a.doc.HeadingIndex(id))
		shot("preview-" + id + ".png")
	}
	a.dialog = NewFileDialog("Open Markdown file", "../mdmodel/testdata", "", false, mdExts, func(string) {})
	shot("dialog.png")
}
