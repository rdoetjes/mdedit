package ui

import (
	"testing"

	"gioui.org/io/key"

	"mdedit/internal/render"
)

func TestThemeToggle(t *testing.T) {
	h := newHarness(t, "")
	if h.a.dark {
		t.Fatal("tests start with the light theme (see TestMain)")
	}
	h.press("D", key.ModShortcut|key.ModShift)
	if !h.a.dark || col != darkColors || h.a.preview.ctx.Theme != &render.Dark {
		t.Fatal("Ctrl/⌘+Shift+D did not switch to the dark theme")
	}
	if h.a.cmds.theme.label != "Light Theme" {
		t.Errorf("menu label %q, want %q", h.a.cmds.theme.label, "Light Theme")
	}
	// The choice is remembered for the next start.
	if loadSettings().Theme != "dark" {
		t.Error("dark theme not saved")
	}
	if next := newHarness(t, ""); !next.a.dark {
		t.Error("the next start isn't dark")
	}
	h.press("D", key.ModShortcut|key.ModShift)
	if h.a.dark || col != lightColors || loadSettings().Theme != "light" {
		t.Error("switching back to light failed")
	}
}
