//go:build !js

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestTableDialog(t *testing.T) {
	h := newHarness(t, "")
	h.setText("Intro", 5, 5)
	h.a.tableDialog()
	h.frame()
	d := h.a.dialog.(*TableDialog)
	d.cols.SetText("2")
	d.rows.SetText("1")
	d.submit()
	h.frame()
	h.frame()
	h.expect("Intro\n\n|     |     |\n| --- | --- |\n|     |     |\n")
	if h.a.dialog != nil {
		t.Error("dialog still open")
	}
}

// pathRecorder records the path a file dialog finishes with.
type pathRecorder struct {
	h    *harness
	path string
}

func (r *pathRecorder) done(p string) {
	r.path = p
	r.h.a.dialog = nil
}

// makeFolders creates n empty folders and a file in the first; it returns
// that first folder.
func makeFolders(t *testing.T, root string, n int) string {
	for i := 0; i < n; i++ {
		if err := os.Mkdir(filepath.Join(root, fmt.Sprintf("d%02d", i)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	first := filepath.Join(root, "d00")
	if err := os.WriteFile(filepath.Join(first, "note.md"), []byte("# hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	return first
}

// Regression: clicking a folder that has fewer entries than the current one
// crashed, because the click loop kept indexing the old entry count.
func TestFileDialogEnterSmallerFolder(t *testing.T) {
	root := t.TempDir()
	small := makeFolders(t, root, 20)
	h := newHarness(t, "")
	rec := &pathRecorder{h: h}
	d := NewFileDialog("Open", root, "", false, mdExts, rec.done)
	h.a.dialog = d
	h.frame()

	d.clicks[0].Click() // enter d00, which has a single entry
	h.frame()
	h.frame()
	if d.dir != small || len(d.entries) != 1 || d.entries[0].Name() != "note.md" {
		t.Fatalf("dialog in %s with %v, want %s with note.md", d.dir, d.entries, small)
	}
	// Up again, then into the folder once more, then open the file.
	d.up.Click()
	h.frame()
	d.clicks[0].Click()
	h.frame()
	d.name.SetText("note.md")
	d.ok.Click()
	h.frame()
	if rec.path != filepath.Join(small, "note.md") {
		t.Errorf("picked %q", rec.path)
	}
}

// The Finder's "Open With" arrives as an open request from another
// goroutine; the file opens on the next frame.
func TestOpenRequest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(path, []byte("# From Finder\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, "")
	done := make(chan bool)
	go requestOpen(h.a, path, done)
	<-done
	h.frame()
	h.expect("# From Finder\n")
	if h.a.path != path {
		t.Errorf("path %q, want %q", h.a.path, path)
	}
}

func requestOpen(a *App, path string, done chan bool) {
	a.RequestOpen(path)
	done <- true
}
