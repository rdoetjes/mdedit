package ui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// maxRecent is the number of recently used files in the File menu.
const maxRecent = 5

// addRecent puts p at the front of the list, without duplicates, keeping
// at most maxRecent entries.
func addRecent(list []string, p string) []string {
	out := []string{p}
	for _, q := range list {
		if q != p && len(out) < maxRecent {
			out = append(out, q)
		}
	}
	return out
}

// rememberRecent adds a file just opened or saved to the recent files.
func (a *App) rememberRecent(p string) {
	if !recentFilesSupported || p == "" {
		return
	}
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	a.setRecent(addRecent(a.recent, p))
}

// forgetRecent removes a file that no longer exists.
func (a *App) forgetRecent(p string) {
	a.setRecent(slices.DeleteFunc(slices.Clone(a.recent), func(q string) bool { return q == p }))
}

// setRecent changes the list and saves it; the File menu shows it the
// next time it opens.
func (a *App) setRecent(list []string) {
	if slices.Equal(list, a.recent) {
		return
	}
	a.recent = list
	st := loadSettings()
	st.Recent = list
	a.saveSettings(st, "the recent files")
}

// recentCommands are the File menu entries for the recent files.
func (a *App) recentCommands() []*command {
	var out []*command
	for _, p := range a.recent {
		out = append(out, newCommand(recentLabel(p), "", 0, func(C) { a.openRecent(p) }))
	}
	return out
}

// recentLabel shows a file as its name and folder, the home folder as ~.
func recentLabel(p string) string {
	dir := filepath.Dir(p)
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if rel, err := filepath.Rel(home, dir); err == nil && !strings.HasPrefix(rel, "..") {
			dir = filepath.Join("~", rel)
		}
	}
	return filepath.Base(p) + "  —  " + dir
}

// openRecent opens a recent file, or removes it from the list if it is gone.
func (a *App) openRecent(p string) {
	if _, err := os.Stat(p); err != nil {
		a.setStatus("%s no longer exists; removed from the recent files", filepath.Base(p))
		a.forgetRecent(p)
		return
	}
	a.flow.linkPath = p
	a.confirmDiscard(actOpenFile)
}
