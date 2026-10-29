//go:build !js

package ui

import (
	"testing"
	"time"

	"gioui.org/layout"

	"mdedit/internal/spell"
	"mdedit/internal/vfs"
)

// A tiny stand-in for the Dutch dictionary, so the test needs no network.
const (
	fakeDutchAff = "SET UTF-8\nSFX A Y 1\nSFX A 0 en .\n"
	fakeDutchDic = "2\nfiets/A\nhuis\n"
)

// Choosing a language that is on disk loads it in the background and
// checks with it as well as English.
func TestSpellingLanguages(t *testing.T) {
	if err := vfs.WriteFile(dictPath("nl", "aff"), []byte(fakeDutchAff)); err != nil {
		t.Fatal(err)
	}
	if err := vfs.WriteFile(dictPath("nl", "dic"), []byte(fakeDutchDic)); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, "")
	h.setText("The fietsen are here, wrold.\n", 0, 0)
	if n := len(h.a.spell.misses); n != 2 {
		t.Fatalf("%d misses before choosing Dutch, want 2", n)
	}
	h.a.cmdLanguages(layout.Context{})
	d := h.a.dialog.(*LanguageDialog)
	if !d.have[languageIndex("nl")] {
		t.Error("the dialog doesn't see the Dutch dictionary on disk")
	}
	d.checks[languageIndex("nl")].Value = true
	d.done(d.chosen())
	h.waitFor(dutchLoaded)
	if n := len(h.a.spell.misses); n != 1 || h.a.spell.misses[0].Word != "wrold" {
		t.Errorf("misses %v, want just wrold", h.a.spell.misses)
	}
	if got := loadSettings().Languages; len(got) != 2 {
		t.Errorf("saved languages %q", got)
	}
}

func dutchLoaded(a *App) bool { return a.spell.loaded["nl"] != nil }

// waitFor runs frames until ready reports true, for up to 5 seconds.
func (h *harness) waitFor(ready func(*App) bool) {
	h.t.Helper()
	for deadline := time.Now().Add(5 * time.Second); !ready(h.a); {
		if time.Now().After(deadline) {
			h.t.Fatal("timed out")
		}
		time.Sleep(10 * time.Millisecond)
		h.frame()
	}
}

func languageIndex(code string) int {
	for i, l := range spell.Languages {
		if l.Code == code {
			return i
		}
	}
	return -1
}
