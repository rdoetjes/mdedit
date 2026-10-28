// Package imgcache downloads remote images into an on-disk cache. Only
// YouTube video stills are fetched, so documents can't make mdedit
// download arbitrary URLs.
package imgcache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif" // decoders for checking downloads
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mdedit/internal/youtube"
)

const (
	maxSize    = 10 << 20 // bytes per image
	retryAfter = time.Minute
)

var (
	client = &http.Client{Timeout: 15 * time.Second}

	// allowed and fallbackFor are variables so tests can serve images
	// locally.
	allowed     = youtube.IsThumbnail
	fallbackFor = youtubeFallback

	mu       sync.Mutex
	inflight = map[string]bool{}
	failed   = map[string]time.Time{}
	dir      string
)

// ErrNotAllowed is returned for images mdedit doesn't download.
var ErrNotAllowed = errors.New("remote image not allowed")

// Allowed reports whether src is a remote image mdedit downloads.
func Allowed(src string) bool { return allowed(src) }

// Path returns the cached file of a downloaded image, if there is one.
func Path(src string) (string, bool) {
	p := file(src)
	if _, err := os.Stat(p); err != nil {
		return "", false
	}
	return p, true
}

// Fetch downloads src in the background, unless it is cached, already
// being downloaded or recently failed. done is called after a successful
// download.
func Fetch(src string, done func()) {
	if !allowed(src) || !start(src) {
		return
	}
	go fetchAsync(src, done)
}

func fetchAsync(src string, done func()) {
	err := download(src)
	finish(src, err)
	if err == nil && done != nil {
		done()
	}
}

// Get returns the cached file of src, downloading it first if needed.
func Get(src string) (string, error) {
	if !allowed(src) {
		return "", ErrNotAllowed
	}
	if p, ok := Path(src); ok {
		return p, nil
	}
	if recentlyFailed(src) {
		return "", ErrRecentFailure
	}
	err := download(src)
	finish(src, err)
	if err != nil {
		return "", err
	}
	return file(src), nil
}

// ErrRecentFailure is returned while a failed download isn't retried yet.
var ErrRecentFailure = errors.New("download failed recently")

func recentlyFailed(src string) bool {
	mu.Lock()
	defer mu.Unlock()
	return time.Since(failed[src]) < retryAfter
}

// start marks src as being downloaded; it reports false if there's no
// need to.
func start(src string) bool {
	if _, ok := Path(src); ok {
		return false
	}
	mu.Lock()
	defer mu.Unlock()
	if inflight[src] || time.Since(failed[src]) < retryAfter {
		return false
	}
	inflight[src] = true
	return true
}

func finish(src string, err error) {
	mu.Lock()
	defer mu.Unlock()
	delete(inflight, src)
	if err != nil {
		failed[src] = time.Now()
	}
}

// download fetches src (or its fallback) and stores it in the cache.
func download(src string) error {
	data, err := fetch(src)
	if errors.Is(err, errNotFound) {
		if fb := fallbackFor(src); fb != "" {
			data, err = fetch(fb)
		}
	}
	if err != nil {
		return err
	}
	return store(file(src), data)
}

var errNotFound = errors.New("not found")

// fetch downloads an image and checks that it decodes.
func fetch(src string) ([]byte, error) {
	resp, err := client.Get(src)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", src, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return nil, err
	}
	if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("%s: not an image: %w", src, err)
	}
	return data, nil
}

// store writes the file atomically, so readers never see a partial image.
func store(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// youtubeFallback is the still that exists for every video, for videos
// without a full-resolution one.
func youtubeFallback(src string) string {
	id := youtube.ThumbnailID(src)
	if id == "" || src == youtube.FallbackThumbnailURL(id) {
		return ""
	}
	return youtube.FallbackThumbnailURL(id)
}

// file is the cache file for src, named by a hash of the URL and keeping
// its extension, which the PDF writer uses to tell the image type.
func file(src string) string {
	sum := sha256.Sum256([]byte(src))
	return filepath.Join(cacheDir(), hex.EncodeToString(sum[:16])+filepath.Ext(src))
}

func cacheDir() string {
	mu.Lock()
	defer mu.Unlock()
	if dir == "" {
		dir = defaultDir()
	}
	return dir
}

func defaultDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "mdedit", "images")
}
