package ui

import (
	"os"
	"path/filepath"
	"strings"

	"gioui.org/io/system"

	"mdedit/internal/pdfout"
)

// action is something to do once unsaved changes have been dealt with, or
// once the document has been saved.
type action int

const (
	actNone action = iota
	actNew
	actOpenDialog
	actOpenFile // opens flow.linkPath
	actQuit
)

// target says what to write once overwriting a file has been confirmed.
type target int

const (
	targetMarkdown target = iota
	targetPDF
)

// flow holds the state of multi-step file operations, which pass through
// dialogs: what to do after discarding or saving, and where to write.
type flow struct {
	afterDiscard action
	afterSave    action
	linkPath     string
	dest         string
	destKind     target
}

// perform runs an action.
func (a *App) perform(act action) {
	switch act {
	case actNew:
		a.load("", "")
		a.setStatus("New document")
	case actOpenDialog:
		a.showOpenDialog()
	case actOpenFile:
		a.openFile(a.flow.linkPath)
	case actQuit:
		a.dirty = false
		a.win.Perform(system.ActionClose)
	}
}

// confirmDiscard performs act, after asking to save unsaved changes first.
func (a *App) confirmDiscard(act action) {
	if !a.dirty {
		a.perform(act)
		return
	}
	a.flow.afterDiscard = act
	a.dialog = &ConfirmDialog{
		Message: "The document has unsaved changes. Save them first?",
		Buttons: []string{"Cancel", "Don't save", "Save"},
		Done:    a.discardAnswered,
	}
}

func (a *App) discardAnswered(choice int) {
	a.dialog = nil
	switch choice {
	case 1:
		a.dirty = false
		a.perform(a.flow.afterDiscard)
	case 2:
		a.save(a.flow.afterDiscard)
	}
}

func (a *App) newDoc()     { a.confirmDiscard(actNew) }
func (a *App) openDialog() { a.confirmDiscard(actOpenDialog) }
func (a *App) quit()       { a.confirmDiscard(actQuit) }

func (a *App) openChosen(p string) {
	a.dialog = nil
	if p != "" {
		a.openFile(p)
	}
}

func (a *App) openFile(p string) {
	data, err := os.ReadFile(p)
	if err != nil {
		a.setStatus("Open failed: %v", err)
		return
	}
	a.load(p, string(data))
	a.setStatus("Opened %s", p)
}

func (a *App) saveAsChosen(p string) {
	a.dialog = nil
	if p != "" {
		a.confirmOverwrite(p, targetMarkdown, p != a.path)
	}
}

func (a *App) exportChosen(p string) {
	a.dialog = nil
	if p != "" {
		a.confirmOverwrite(p, targetPDF, true)
	}
}

// confirmOverwrite writes dest, asking first if it exists and ask is set.
func (a *App) confirmOverwrite(dest string, kind target, ask bool) {
	a.flow.dest, a.flow.destKind = dest, kind
	if _, err := os.Stat(dest); err != nil || !ask {
		a.writeDest()
		return
	}
	a.dialog = &ConfirmDialog{
		Message: filepath.Base(dest) + " already exists. Replace it?",
		Buttons: []string{"Cancel", "Replace"},
		Done:    a.overwriteAnswered,
	}
}

func (a *App) overwriteAnswered(choice int) {
	a.dialog = nil
	if choice == 1 {
		a.writeDest()
	}
}

func (a *App) writeDest() {
	if a.flow.destKind == targetPDF {
		a.exportPDF(a.flow.dest)
	} else {
		a.writeFile(a.flow.dest, a.flow.afterSave)
	}
}

func (a *App) writeFile(p string, then action) {
	txt := a.editor.Text()
	if err := os.WriteFile(p, []byte(txt), 0o644); err != nil {
		a.setStatus("Save failed: %v", err)
		return
	}
	a.path = p
	a.rememberRecent(p)
	a.saved = txt
	a.dirty = false
	a.preview.BaseDir = filepath.Dir(p)
	a.setStatus("Saved %s", p)
	a.perform(then)
}

func (a *App) exportPDF(p string) {
	if err := a.writePDF(p); err != nil {
		a.setStatus("Export failed: %v", err)
		return
	}
	a.setStatus("Exported %s", p)
}

func (a *App) writePDF(p string) error {
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	err = pdfout.Write(f, a.doc, pdfout.Options{BaseDir: a.preview.BaseDir, Title: a.baseName()})
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// followLink handles a click on a link in the preview.
func (a *App) followLink(url string) {
	switch {
	case strings.HasPrefix(url, "#"):
		a.jumpToHeading(url)
	case strings.Contains(url, ":"):
		a.openExternal(url)
	default:
		a.openRelative(url)
	}
}

func (a *App) jumpToHeading(anchor string) {
	if i := a.doc.HeadingIndex(anchor[1:]); i >= 0 {
		a.preview.ScrollToBlock(i)
		return
	}
	a.setStatus("No heading with anchor %s", anchor)
}

func (a *App) openExternal(target string) {
	if err := openURL(target); err != nil {
		a.setStatus("Could not open %s: %v", target, err)
	}
}

// openRelative opens a relative link: Markdown files in the editor, anything
// else with the system's handler.
func (a *App) openRelative(url string) {
	p, _, _ := strings.Cut(url, "#")
	if !filepath.IsAbs(p) {
		p = filepath.Join(a.preview.BaseDir, p)
	}
	if _, err := os.Stat(p); err != nil {
		a.setStatus("Link target not found: %s", url)
		return
	}
	if isMarkdown(p) {
		a.flow.linkPath = p
		a.confirmDiscard(actOpenFile)
		return
	}
	a.openExternal(p)
}

func isMarkdown(p string) bool {
	ext := strings.ToLower(filepath.Ext(p))
	for _, x := range mdExts {
		if ext == x {
			return true
		}
	}
	return false
}
