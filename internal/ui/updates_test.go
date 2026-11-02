//go:build !js

package ui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewerVersion(t *testing.T) {
	tests := []struct {
		tag, current string
		want         bool
	}{
		{"v0.5.0", "0.4.0", true},
		{"v0.10.0", "0.9.0", true}, // numbers, not text
		{"v1.0.0", "0.99.0", true},
		{"v0.4.0", "0.4.0", false},
		{"v0.3.0", "0.4.0", false},
		{"v0.5.0", "dev", false}, // a development build
		{"nightly", "0.4.0", false},
		{"v0.5", "0.4.0", false},
	}
	for _, tt := range tests {
		if got := newerVersion(tt.tag, tt.current); got != tt.want {
			t.Errorf("newerVersion(%q, %q) = %v", tt.tag, tt.current, got)
		}
	}
}

// fakeGitHub answers the latest-release request; status 0 means no answer
// at all, as when offline.
func fakeGitHub(t *testing.T, status int, tag, page string) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		fmt.Fprintf(w, `{"tag_name": %q, "html_url": %q}`, tag, page)
	}))
	latestURL = srv.URL
	if status == 0 {
		srv.Close() // nothing listens: like being offline
	} else {
		t.Cleanup(srv.Close)
	}
	t.Cleanup(func() { latestURL = releasesAPI })
}

// asVersion runs a test as released version v.
func asVersion(t *testing.T, v string) {
	old := Version
	Version = v
	t.Cleanup(func() { Version = old })
}

// startChecking starts mdedit and waits for its check at start-up.
func startChecking(t *testing.T) *harness {
	h := newHarness(t, "")
	h.waitFor(checkDone)
	h.frame()
	return h
}

func checkDone(a *App) bool { return a.updates.arrival == nil }

func TestUpdateOffered(t *testing.T) {
	asVersion(t, "0.4.0")
	page := releasesPage + "tag/v0.5.0"
	fakeGitHub(t, http.StatusOK, "v0.5.0", page)
	var opened string
	openPage = func(u string) error { opened = u; return nil }
	t.Cleanup(func() { openPage = openURL })
	h := startChecking(t)
	d, ok := h.a.dialog.(*ConfirmDialog)
	if !ok {
		t.Fatal("no dialog for a newer version")
	}
	d.Done(1) // Open Release Page
	if opened != page || h.a.dialog != nil {
		t.Errorf("opened %q, dialog %v", opened, h.a.dialog)
	}
}

func TestNoDialogWhenCurrent(t *testing.T) {
	asVersion(t, "0.5.0")
	fakeGitHub(t, http.StatusOK, "v0.5.0", releasesPage+"tag/v0.5.0")
	if h := startChecking(t); h.a.dialog != nil {
		t.Error("a dialog for the version already running")
	}
}

// Offline, an error page, or a page elsewhere: mdedit carries on quietly.
func TestUpdateFailuresAreQuiet(t *testing.T) {
	asVersion(t, "0.4.0")
	cases := []struct {
		name   string
		status int
		page   string
	}{
		{"offline", 0, ""},
		{"error", http.StatusForbidden, ""},
		{"page elsewhere", http.StatusOK, "https://evil.example/mdedit"},
	}
	for _, c := range cases {
		fakeGitHub(t, c.status, "v0.9.0", c.page)
		h := startChecking(t)
		if h.a.dialog != nil || h.a.status != "" {
			t.Errorf("%s: dialog %v, status %q", c.name, h.a.dialog, h.a.status)
		}
	}
}

// The dialog waits until another dialog is closed.
func TestUpdateWaitsForOtherDialog(t *testing.T) {
	asVersion(t, "dev") // no check at start-up: start it below
	fakeGitHub(t, http.StatusOK, "v0.5.0", releasesPage+"tag/v0.5.0")
	h := newHarness(t, "")
	Version = "0.4.0"
	h.a.tableDialog()
	other := h.a.dialog
	h.a.checkForUpdates(false)
	h.waitFor(checkDone)
	h.frame()
	if h.a.dialog != other {
		t.Fatal("the update dialog replaced an open dialog")
	}
	h.a.dialog = nil
	h.frame()
	if _, ok := h.a.dialog.(*ConfirmDialog); !ok {
		t.Error("the update dialog didn't follow")
	}
}

// Turned off, there is no check at start-up; Help → Check for Updates
// still works and says when there is nothing new.
func TestUpdateCheckOptOut(t *testing.T) {
	asVersion(t, "0.5.0")
	fakeGitHub(t, http.StatusOK, "v0.5.0", releasesPage+"tag/v0.5.0")
	h := startChecking(t)
	h.run(h.a.cmds.updateCheck)
	if !loadSettings().NoUpdateCheck {
		t.Fatal("the opt-out wasn't saved")
	}
	next := h.restart()
	if next.a.updates.arrival != nil {
		t.Error("checked at start-up although turned off")
	}
	next.a.cmdCheckUpdates(C{})
	deadline := time.Now().Add(5 * time.Second)
	for next.a.updates.arrival != nil && time.Now().Before(deadline) {
		next.frame()
	}
	if next.a.status != "mdedit 0.5.0 is the latest version" {
		t.Errorf("status %q", next.a.status)
	}
}

// Development builds don't check at start-up.
func TestNoCheckForDevBuilds(t *testing.T) {
	asVersion(t, "dev")
	fakeGitHub(t, http.StatusOK, "v0.5.0", releasesPage+"tag/v0.5.0")
	if h := newHarness(t, ""); h.a.updates.arrival != nil {
		t.Error("a development build checked for updates")
	}
}
