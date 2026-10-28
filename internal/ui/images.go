package ui

import (
	"image"
	"os"

	"gioui.org/op/paint"

	"mdedit/internal/imgcache"
	"mdedit/internal/render"
)

// imageEntry caches a decoded image, or the failure to decode one, for a
// file with the given modification time.
type imageEntry struct {
	mod    int64
	failed bool
	op     paint.ImageOp
	size   image.Point
}

// image returns the decoded image referenced by src, or nil if it can't be
// shown (yet: a YouTube still is downloaded in the background first).
func (p *Preview) image(src string) *imageEntry {
	path, ok := render.ResolveImage(p.BaseDir, src)
	if !ok {
		if imgcache.Allowed(src) {
			imgcache.Fetch(src, p.imageArrived)
		}
		return nil
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return nil
	}
	e := p.cachedImage(path, st.ModTime().UnixNano())
	if e.failed {
		return nil
	}
	return e
}

// cachedImage returns the cache entry for path, decoding the file if it
// changed since it was cached.
func (p *Preview) cachedImage(path string, mod int64) *imageEntry {
	if e, ok := p.images[path]; ok && e.mod == mod {
		return e
	}
	e := decodeImage(path)
	e.mod = mod
	p.images[path] = e
	return e
}

// imageArrived runs when a background download finishes: lay out again
// with the new image.
func (p *Preview) imageArrived() {
	p.imagesChanged.Store(true)
	if p.Invalidate != nil {
		p.Invalidate()
	}
}

func decodeImage(path string) *imageEntry {
	f, err := os.Open(path)
	if err != nil {
		return &imageEntry{failed: true}
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return &imageEntry{failed: true}
	}
	return &imageEntry{op: paint.NewImageOp(img), size: img.Bounds().Size()}
}
