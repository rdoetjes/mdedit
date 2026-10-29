//go:build !js

package ui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestAddRecent(t *testing.T) {
	list := []string{"a", "b", "c", "d", "e"}
	if got := addRecent(list, "c"); !slices.Equal(got, []string{"c", "a", "b", "d", "e"}) {
		t.Errorf("reopening c: %q", got)
	}
	if got := addRecent(list, "f"); !slices.Equal(got, []string{"f", "a", "b", "c", "d"}) {
		t.Errorf("a sixth file: %q", got)
	}
}

// writeDocs creates n Markdown files and returns their paths.
func writeDocs(t *testing.T, n int) []string {
	dir := t.TempDir()
	var paths []string
	for i := range n {
		p := filepath.Join(dir, string(rune('a'+i))+".md")
		if err := os.WriteFile(p, []byte("# "+filepath.Base(p)+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	return paths
}

// recentItems returns the recent file entries of the File menu.
func (h *harness) recentItems() []*command {
	var out []*command
	for _, c := range h.a.menus[0].list() {
		if c != nil && strings.Contains(c.label, "  —  ") {
			out = append(out, c)
		}
	}
	return out
}

// Opening files lists the last five in the File menu, newest first, and
// remembers them for the next start.
func TestRecentFiles(t *testing.T) {
	h := newHarness(t, "")
	if len(h.recentItems()) != 0 {
		t.Fatal("a fresh start has recent files")
	}
	paths := writeDocs(t, 6)
	for _, p := range paths {
		h.a.openFile(p)
	}
	items := h.recentItems()
	if len(items) != maxRecent || !strings.HasPrefix(items[0].label, "f.md") || !strings.HasPrefix(items[4].label, "b.md") {
		t.Fatalf("recent entries %q", labels(items))
	}
	if next := h.restart(); len(next.recentItems()) != maxRecent {
		t.Error("the recent files are not remembered")
	}
	h.run(items[3]) // c.md
	h.expect("# c.md\n")
	if got := h.recentItems()[0].label; !strings.HasPrefix(got, "c.md") {
		t.Errorf("the reopened file is not first: %q", got)
	}
}

// A recent file with unsaved changes in the editor asks first.
func TestRecentFileAsksToSave(t *testing.T) {
	h := newHarness(t, "")
	paths := writeDocs(t, 2)
	h.a.openFile(paths[0])
	h.a.openFile(paths[1])
	h.setText("changed", 0, 0)
	h.a.dirty = true
	h.run(h.recentItems()[1])
	if _, asks := h.a.dialog.(*ConfirmDialog); !asks {
		t.Fatal("opening a recent file did not ask about the unsaved changes")
	}
}

// A recent file that was deleted is removed from the list.
func TestRecentFileGone(t *testing.T) {
	h := newHarness(t, "")
	paths := writeDocs(t, 2)
	h.a.openFile(paths[0])
	h.a.openFile(paths[1])
	os.Remove(paths[0])
	h.run(h.recentItems()[1])
	if items := h.recentItems(); len(items) != 1 || !strings.HasPrefix(items[0].label, "b.md") {
		t.Errorf("recent entries %q, want only b.md", labels(items))
	}
	h.expect("# b.md\n")
}
