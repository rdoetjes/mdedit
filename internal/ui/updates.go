package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gioui.org/layout"
)

// The update check asks GitHub, once at start-up and in the background,
// for the latest release. If it is newer than this version, a dialog offers
// its page; offline, or on any error, nothing happens. This is how users of
// the desktop version learn about security updates (Cyber Resilience Act,
// Annex I, Part I, 2(c)); the browser version is updated by its site.
// Help → Check for Updates checks now; the check at start-up can be turned
// off, since it reveals the user's address to GitHub.

const (
	// releasesAPI is the latest release of the repository, on GitHub.
	releasesAPI = "https://api.github.com/repos/rdoetjes/mdedit/releases/latest"
	// releasesPage is where release pages must be, to be opened.
	releasesPage  = "https://github.com/rdoetjes/mdedit/releases/"
	updateTimeout = 10 * time.Second
)

// latestURL is releasesAPI, except in tests.
var latestURL = releasesAPI

// release is what the check found.
type release struct {
	Tag  string `json:"tag_name"`
	Page string `json:"html_url"`
	err  error
}

// fetchLatest asks for the latest release; it runs on its own goroutine.
func fetchLatest(out chan<- release, wake func()) {
	defer wake()
	client := &http.Client{Timeout: updateTimeout}
	req, err := http.NewRequest(http.MethodGet, latestURL, nil)
	if err != nil {
		out <- release{err: err}
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "mdedit/"+Version)
	resp, err := client.Do(req)
	if err != nil {
		out <- release{err: err}
		return
	}
	defer resp.Body.Close()
	var r release
	if resp.StatusCode != http.StatusOK {
		r.err = fmt.Errorf("GitHub answered %s", resp.Status)
	} else {
		r.err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&r)
	}
	out <- r
}

// updateChecker carries a check from its start to the dialog.
type updateChecker struct {
	arrival chan release
	manual  bool     // asked for in the menu: also say "up to date"
	pending *release // a newer release, waiting for other dialogs to close
}

// checkForUpdates starts a check, unless one is running.
func (a *App) checkForUpdates(manual bool) {
	u := &a.updates
	if u.arrival != nil {
		return
	}
	u.manual = manual
	u.arrival = make(chan release, 1)
	go fetchLatest(u.arrival, a.invalidate)
}

// handleUpdate takes a finished check, and shows its dialog once no other
// dialog is open.
func (a *App) handleUpdate() {
	u := &a.updates
	select {
	case r := <-u.arrival:
		u.arrival = nil
		a.updateArrived(r)
	default:
	}
	if u.pending != nil && a.dialog == nil {
		a.offerUpdate(*u.pending)
		u.pending = nil
	}
}

func (a *App) updateArrived(r release) {
	u := &a.updates
	switch {
	case r.err != nil:
		if u.manual {
			a.setStatus("Could not check for updates: %v", r.err)
		}
	case newerVersion(r.Tag, Version) && strings.HasPrefix(r.Page, releasesPage):
		u.pending = &r
	case u.manual:
		a.setStatus("mdedit %s is the latest version", Version)
	}
}

// offerUpdate shows the dialog for a newer release.
func (a *App) offerUpdate(r release) {
	a.dialog = &ConfirmDialog{
		Message: fmt.Sprintf("mdedit %s is available; you have %s. New versions can contain security "+
			"fixes. Open the release page to download it?", strings.TrimPrefix(r.Tag, "v"), Version),
		Buttons: []string{"Later", "Open Release Page"},
		Done:    func(choice int) { a.updateAnswered(r, choice) },
	}
}

func (a *App) updateAnswered(r release, choice int) {
	a.dialog = nil
	a.focusEditor = true
	if choice == 1 {
		if err := openPage(r.Page); err != nil {
			a.setStatus("Could not open %s: %v", r.Page, err)
		}
	}
}

// newerVersion reports whether tag (v1.2.3) is a later version than
// current (1.2.3). A current version that isn't a version, such as "dev",
// is never older.
func newerVersion(tag, current string) bool {
	t, ok1 := versionParts(strings.TrimPrefix(tag, "v"))
	c, ok2 := versionParts(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := range t {
		if t[i] != c[i] {
			return t[i] > c[i]
		}
	}
	return false
}

// versionParts splits major.minor.patch.
func versionParts(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// openPage opens a release page; tests replace it.
var openPage = openURL

// startUpdateCheck runs the check at start-up, unless turned off, for a
// released version on the desktop.
func (a *App) startUpdateCheck(st settings) {
	a.setUpdateCheck(!st.NoUpdateCheck)
	if updateCheckSupported && !st.NoUpdateCheck {
		if _, ok := versionParts(Version); ok {
			a.checkForUpdates(false)
		}
	}
}

func (a *App) setUpdateCheck(on bool) {
	a.cmds.updateCheck.label = "Check for Updates at Start"
	if on {
		a.cmds.updateCheck.label = "Don't Check for Updates at Start"
	}
}

// cmdCheckUpdates checks now, from the menu.
func (a *App) cmdCheckUpdates(layout.Context) {
	if _, ok := versionParts(Version); !ok {
		a.setStatus("This is a development build (%s): there is nothing to compare", Version)
		return
	}
	a.setStatus("Checking for updates…")
	a.checkForUpdates(true)
}

// cmdToggleUpdateCheck turns the check at start-up on or off.
func (a *App) cmdToggleUpdateCheck(layout.Context) {
	st := loadSettings()
	st.NoUpdateCheck = !st.NoUpdateCheck
	a.setUpdateCheck(!st.NoUpdateCheck)
	a.saveSettings(st, "the update setting")
}
