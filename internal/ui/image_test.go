//go:build !js

package ui

import (
	"os"
	"path/filepath"
	"testing"

	"gioui.org/font/gofont"
	"gioui.org/text"
)

// Regression: an image reference to something that exists but isn't an
// image (a folder, or a half-typed name matching another file) crashed on
// the next frame, because the failure was cached as a nil entry.
func TestImageNotDecodable(t *testing.T) {
	dir := t.TempDir()
	notImage := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(notImage, []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := NewPreview(text.NewShaper(text.WithCollection(gofont.Collection())))
	p.BaseDir = dir
	for _, src := range []string{"notes.txt", ".", "missing.png"} {
		for i := 0; i < 3; i++ { // the second lookup used to panic
			if img := p.image(src); img != nil {
				t.Errorf("%q: got an image", src)
			}
		}
	}
}
