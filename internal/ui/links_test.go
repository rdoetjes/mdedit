//go:build !js

package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeLink(t *testing.T) {
	good := []string{"https://example.com/x", "http://example.com", "mailto:a@example.com", "HTTPS://EXAMPLE.COM"}
	for _, l := range good {
		if !safeLink(l) {
			t.Errorf("%q should open", l)
		}
	}
	bad := []string{"file:///etc/passwd", "smb://host/share", "javascript:alert(1)", "ms-msdt:/id", "vscode://x",
		"tel:123", "/Applications/Calculator.app", "C:\\Windows\\calc.exe", ""}
	for _, l := range bad {
		if safeLink(l) {
			t.Errorf("%q must not open", l)
		}
	}
}

// A crafted document can't reach the operating system's "open" on a click.
func TestLinkToOtherSchemeRefused(t *testing.T) {
	h := newHarness(t, "")
	h.a.followLink("file:///Applications/Calculator.app")
	if h.a.status == "" || h.a.dialog != nil {
		t.Errorf("status %q, dialog %v", h.a.status, h.a.dialog)
	}
}

// A link to a local file that isn't Markdown asks first, and Cancel keeps
// it closed.
func TestLinkToLocalFileAsks(t *testing.T) {
	h := newHarness(t, "")
	dir := t.TempDir()
	p := filepath.Join(dir, "run.command")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	h.a.preview.BaseDir = dir
	h.a.followLink("run.command")
	d, asks := h.a.dialog.(*ConfirmDialog)
	if !asks {
		t.Fatal("no question before opening a local file")
	}
	d.Done(0) // Cancel
	if h.a.dialog != nil {
		t.Error("the dialog stays open after Cancel")
	}
}
