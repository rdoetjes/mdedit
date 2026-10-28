package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func serveDir(t *testing.T) files {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "index.html"), "<html>")
	write(t, filepath.Join(dir, "mdedit.wasm"), "wasm")
	write(t, filepath.Join(dir, "mdedit.wasm.gz"), "gzipped")
	return files{root: dir}
}

func write(t *testing.T, p, s string) {
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func get(f files, url, accept string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, url, nil)
	if accept != "" {
		r.Header.Set("Accept-Encoding", accept)
	}
	w := httptest.NewRecorder()
	f.ServeHTTP(w, r)
	return w
}

func TestCompressedCopy(t *testing.T) {
	w := get(serveDir(t), "/mdedit.wasm", "gzip, deflate, br")
	if w.Body.String() != "gzipped" || w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("got %q encoding %q", w.Body.String(), w.Header().Get("Content-Encoding"))
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/wasm" {
		t.Fatalf("content type %q", ct)
	}
}

func TestPlainWithoutAcceptEncoding(t *testing.T) {
	w := get(serveDir(t), "/mdedit.wasm", "")
	if w.Body.String() != "wasm" || w.Header().Get("Content-Encoding") != "" {
		t.Fatalf("got %q encoding %q", w.Body.String(), w.Header().Get("Content-Encoding"))
	}
}

func TestIndexAndEscape(t *testing.T) {
	f := serveDir(t)
	if w := get(f, "/", ""); w.Body.String() != "<html>" {
		t.Fatalf("/ served %q", w.Body.String())
	}
	if w := get(f, "/../../etc/passwd", ""); w.Code != http.StatusNotFound {
		t.Fatalf("escaping the folder: status %d", w.Code)
	}
}
