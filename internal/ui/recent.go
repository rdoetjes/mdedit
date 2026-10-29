package ui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gioui.org/layout"
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
	a.setRecent(slices.DeleteFunc(slices.Clone(a.recent), sameFile(p).is))
}

type sameFile string

func (f sameFile) is(p string) bool { return p == string(f) }

// setRecent changes the list, saves it and updates the File menu.
func (a *App) setRecent(list []string) {
	if slices.Equal(list, a.recent) {
		return
	}
	a.recent = list
	a.rebuildFileMenu()
	st := loadSettings()
	st.Recent = list
	if err := st.save(); err != nil {
		a.setStatus("Could not save the recent files: %v", err)
	}
}

func (a *App) rebuildFileMenu() {
	a.menus[0] = a.fileMenu()
	a.shortcuts = shortcutCommands(a.menus)
}

// recentCommands are the File menu entries for the recent files.
func (a *App) recentCommands() []*command {
	var out []*command
	for _, p := range a.recent {
		out = append(out, newCommand(recentLabel(p), "", 0, recentFile{a, p}.run))
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

// recentFile opens a recent file when its menu entry is chosen.
type recentFile struct {
	a    *App
	path string
}

func (r recentFile) run(layout.Context) {
	if _, err := os.Stat(r.path); err != nil {
		r.a.setStatus("%s no longer exists; removed from the recent files", filepath.Base(r.path))
		r.a.forgetRecent(r.path)
		return
	}
	r.a.flow.linkPath = r.path
	r.a.confirmDiscard(actOpenFile)
}
