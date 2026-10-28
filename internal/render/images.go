package render

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"mdedit/internal/imgcache"
	"mdedit/internal/vfs"
)

// ResolveImage maps an image reference from a document in baseDir to a local
// file path. Of remote images only YouTube stills are supported; they
// resolve to their downloaded copy once package imgcache has fetched them.
func ResolveImage(baseDir, src string) (string, bool) {
	if imgcache.Allowed(src) {
		return imgcache.Path(src)
	}
	if strings.Contains(src, "://") {
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
	mod, err := vfs.ModTime(path)
	if err != nil {
		return 0, 0, false
	}
	key := path + "\x00" + strconv.FormatInt(mod, 10)
	sizeMu.Lock()
	defer sizeMu.Unlock()
	if v, hit := sizeCache[key]; hit {
		return v[0], v[1], v[2] == 1
	}
	f, err := vfs.Open(path)
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	ok = err == nil
	sizeCache[key] = [3]int{cfg.Width, cfg.Height, map[bool]int{false: 0, true: 1}[ok]}
	return cfg.Width, cfg.Height, ok
}
