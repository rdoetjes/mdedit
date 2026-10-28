package imgcache

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// testServer serves a PNG at /vi/ok/ and /vi/x/fallback.png, and 404
// elsewhere.
func testServer(t *testing.T) *httptest.Server {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 16, 9))); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(pngHandler{data: buf.Bytes()})
	t.Cleanup(srv.Close)
	return srv
}

type pngHandler struct{ data []byte }

func (h pngHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/vi/ok/") || r.URL.Path == "/vi/x/fallback.png" {
		w.Write(h.data)
		return
	}
	http.NotFound(w, r)
}

// useTestServer points the cache at a temporary folder and allows the
// test server's URLs.
func useTestServer(t *testing.T, srv *httptest.Server) {
	t.Cleanup(saved{allowed: allowed, fallbackFor: fallbackFor, dir: dir}.restore)
	dir = t.TempDir()
	allowed = (prefix(srv.URL)).has
	fallbackFor = (fallback(srv.URL + "/vi/x/fallback.png")).to
	inflight, failed = map[string]bool{}, map[string]time.Time{}
}

// saved holds the package settings a test changes.
type saved struct {
	allowed     func(string) bool
	fallbackFor func(string) string
	dir         string
}

func (s saved) restore() { allowed, fallbackFor, dir = s.allowed, s.fallbackFor, s.dir }

type prefix string

func (p prefix) has(s string) bool { return strings.HasPrefix(s, string(p)) }

type fallback string

func (f fallback) to(string) string { return string(f) }

func TestGetAndFallback(t *testing.T) {
	srv := testServer(t)
	useTestServer(t, srv)
	p, err := Get(srv.URL + "/vi/ok/maxresdefault.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	// A missing full-resolution still falls back to the other one.
	if _, err := Get(srv.URL + "/vi/missing/maxresdefault.png"); err != nil {
		t.Errorf("fallback: %v", err)
	}
	if _, err := Get("https://example.com/x.png"); err != ErrNotAllowed {
		t.Errorf("disallowed URL: %v", err)
	}
}

func TestFetchAsync(t *testing.T) {
	srv := testServer(t)
	useTestServer(t, srv)
	src := srv.URL + "/vi/ok/still.png"
	done := make(chan bool, 1)
	Fetch(src, signal(done).send)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("download did not finish")
	}
	if _, ok := Path(src); !ok {
		t.Error("not cached after Fetch")
	}
}

type signal chan bool

func (s signal) send() { s <- true }

// A failed download isn't retried straight away, so an offline PDF export
// doesn't wait for every attempt.
func TestFailureNotRetried(t *testing.T) {
	srv := testServer(t)
	useTestServer(t, srv)
	fallbackFor = noFallback
	src := srv.URL + "/vi/missing/still.png"
	if _, err := Get(src); err == nil {
		t.Fatal("download of a missing image succeeded")
	}
	if _, err := Get(src); err != ErrRecentFailure {
		t.Errorf("second attempt: %v, want ErrRecentFailure", err)
	}
}

func noFallback(string) string { return "" }
