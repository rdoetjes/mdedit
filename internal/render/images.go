package render

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ResolveImage maps an image reference from a document in baseDir to a local
// file path. Remote images are not supported.
func ResolveImage(baseDir, src string) (string, bool) {
	if i := strings.Index(src, "://"); i >= 0 {
		if !strings.HasPrefix(src, "file://") {
			return "", false
		}
		src = strings.TrimPrefix(src, "file://")
	}
	if p, err := url.PathUnescape(src); err == nil {
		src = p
	}
	if !filepath.IsAbs(src) {
		src = filepath.Join(baseDir, src)
	}
	return src, true
}

var (
	sizeMu    sync.Mutex
	sizeCache = map[string][3]int{}
)

// ImageFileSize returns the pixel size of a PNG, JPEG or GIF file.
func ImageFileSize(path string) (w, h int, ok bool) {
	st, err := os.Stat(path)
	if err != nil {
		return 0, 0, false
	}
	key := path + "\x00" + st.ModTime().String()
	sizeMu.Lock()
	defer sizeMu.Unlock()
	if v, hit := sizeCache[key]; hit {
		return v[0], v[1], v[2] == 1
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	ok = err == nil
	sizeCache[key] = [3]int{cfg.Width, cfg.Height, map[bool]int{false: 0, true: 1}[ok]}
	return cfg.Width, cfg.Height, ok
}
