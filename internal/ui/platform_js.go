//go:build js

package ui

// The browser side of everything that differs between the desktop
// (platform_desktop.go) and the browser. A web page has no file system:
// Open uses the browser's file picker, Save and Export PDF download the
// file, and the text and settings are kept in the browser's localStorage,
// so a reload or a closed tab loses nothing.

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"syscall/js"

	"gioui.org/font"
	"gioui.org/font/opentype"

	"mdedit/internal/mdmodel"
	"mdedit/internal/pdfout"
	"mdedit/internal/tex"
)

// localStorage keys.
const (
	keyText     = "mdedit.text"     // the editor's text, kept on every change
	keySaved    = "mdedit.saved"    // the text as last opened or downloaded
	keyName     = "mdedit.name"     // the document's file name
	keySettings = "mdedit.settings" // settings.json's contents
)

// platformState carries results from the browser's callbacks and from
// background exports to the frame loop.
type platformState struct {
	mu        sync.Mutex
	opened    *pickedFile // a file chosen in the picker, not yet loaded
	status    string      // a message from a background job
	pickName  string      // name of the file being read
	exporting bool

	ready            bool
	onChange, onText js.Func
}

// pickedFile is a document read from the browser's file picker.
type pickedFile struct{ name, text string }

// platformFonts adds the maths font, for symbols the Go fonts lack (⌘, ⇧,
// arrows); a web page has no system fonts to fall back on.
func platformFonts() []font.FontFace {
	faces, err := opentype.ParseCollection(tex.MathFont())
	if err != nil {
		return nil
	}
	return faces
}

// handleArrivals loads a picked file and shows messages from background
// jobs; it runs every frame.
func (a *App) handleArrivals() {
	p := &a.platform
	p.mu.Lock()
	opened, status := p.opened, p.status
	p.opened, p.status = nil, ""
	p.mu.Unlock()
	if opened != nil {
		a.load(opened.name, opened.text)
		a.setStatus("Opened %s", opened.name)
	}
	if status != "" {
		a.setStatus("%s", status)
	}
}

// deliver hands a message from another goroutine to the frame loop.
func (a *App) deliver(status string) {
	a.platform.mu.Lock()
	a.platform.status = status
	a.platform.mu.Unlock()
	a.invalidate()
}

// autosave keeps the text in the browser after every change.
func (a *App) autosave() {
	err := setItem(keyText, a.editor.Text())
	if err == nil {
		err = setItem(keySaved, a.saved)
	}
	if err == nil {
		err = setItem(keyName, a.path)
	}
	if err != nil {
		a.setStatus("The browser could not keep the text: %v", err)
	}
}

// quitCommands: a web page is closed with its tab, so there is no Quit.
func (a *App) quitCommands() []*command { return nil }

// loadInitial restores the text from the last visit.
func (a *App) loadInitial(string) error {
	// Read everything first: load autosaves, which overwrites the keys.
	name, _ := getItem(keyName)
	saved, _ := getItem(keySaved)
	text, hasText := getItem(keyText)
	a.load(name, saved)
	if hasText && text != saved {
		a.editor.SetText(text)
		a.reparse()
	}
	return nil
}

// saveRecovery isn't needed: autosave has kept the text all along.
func (a *App) saveRecovery() {}

// showOpenDialog shows the browser's file picker; fileChosen and textRead
// receive the result.
func (a *App) showOpenDialog() {
	a.initCallbacks()
	input := js.Global().Get("document").Call("createElement", "input")
	input.Set("type", "file")
	input.Set("accept", strings.Join(mdExts, ","))
	input.Call("addEventListener", "change", a.platform.onChange)
	input.Call("click")
}

func (a *App) initCallbacks() {
	if a.platform.ready {
		return
	}
	a.platform.onChange = js.FuncOf(a.fileChosen)
	a.platform.onText = js.FuncOf(a.textRead)
	a.platform.ready = true
}

// fileChosen runs when a file was picked: it starts reading it.
func (a *App) fileChosen(_ js.Value, args []js.Value) any {
	files := args[0].Get("target").Get("files")
	if files.Length() == 0 {
		return nil
	}
	f := files.Index(0)
	a.platform.mu.Lock()
	a.platform.pickName = f.Get("name").String()
	a.platform.mu.Unlock()
	f.Call("text").Call("then", a.platform.onText)
	return nil
}

// textRead runs when the picked file has been read.
func (a *App) textRead(_ js.Value, args []js.Value) any {
	p := &a.platform
	p.mu.Lock()
	p.opened = &pickedFile{name: p.pickName, text: args[0].String()}
	p.mu.Unlock()
	a.invalidate()
	return nil
}

// save downloads the document as a Markdown file, then performs `then`.
func (a *App) save(then action) {
	name := a.baseName() + ".md"
	txt := a.editor.Text()
	download(name, "text/markdown", []byte(txt))
	a.path = name
	a.saved = txt
	a.dirty = false
	a.autosave()
	a.setStatus("Downloaded %s", name)
	a.perform(then)
}

// saveAs is Save: where a download goes is the browser's business.
func (a *App) saveAs(then action) { a.save(then) }

// exportDialog makes the PDF in the background (it may have to download
// video stills first) and downloads it.
func (a *App) exportDialog() {
	p := &a.platform
	p.mu.Lock()
	busy := p.exporting
	p.exporting = true
	p.mu.Unlock()
	if busy {
		return
	}
	a.setStatus("Exporting PDF…")
	go a.exportInBackground(a.doc, a.preview.BaseDir, a.baseName())
}

func (a *App) exportInBackground(doc *mdmodel.Document, base, title string) {
	var buf bytes.Buffer
	err := pdfout.Write(&buf, doc, pdfout.Options{BaseDir: base, Title: title})
	a.platform.mu.Lock()
	a.platform.exporting = false
	a.platform.mu.Unlock()
	if err != nil {
		a.deliver(fmt.Sprintf("Export failed: %v", err))
		return
	}
	download(title+".pdf", "application/pdf", buf.Bytes())
	a.deliver("Downloaded " + title + ".pdf")
}

// openURL opens a link in a new browser tab.
func openURL(url string) error {
	js.Global().Call("open", url, "_blank", "noopener")
	return nil
}

func readSettings() ([]byte, error) {
	s, ok := getItem(keySettings)
	if !ok {
		return nil, fmt.Errorf("no settings")
	}
	return []byte(s), nil
}

func writeSettings(data []byte) error { return setItem(keySettings, string(data)) }

// download hands data to the browser as a file download.
func download(name, mime string, data []byte) {
	arr := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(arr, data)
	blob := js.Global().Get("Blob").New([]any{arr}, map[string]any{"type": mime})
	url := js.Global().Get("URL").Call("createObjectURL", blob)
	doc := js.Global().Get("document")
	link := doc.Call("createElement", "a")
	link.Set("href", url)
	link.Set("download", name)
	doc.Get("body").Call("appendChild", link)
	link.Call("click")
	link.Call("remove")
	js.Global().Call("setTimeout", revokeURL, 60000, url) // after the download has started
}

// revokeURL frees a download's data.
var revokeURL = js.FuncOf(revoke)

func revoke(_ js.Value, args []js.Value) any {
	js.Global().Get("URL").Call("revokeObjectURL", args[0])
	return nil
}

func getItem(key string) (string, bool) {
	v := js.Global().Get("localStorage").Call("getItem", key)
	if v.IsNull() || v.IsUndefined() {
		return "", false
	}
	return v.String(), true
}

// setItem stores a value; a full storage (the quota is a few MB) throws,
// which becomes an error.
func setItem(key, value string) (err error) {
	defer recoverJS(&err)
	js.Global().Get("localStorage").Call("setItem", key, value)
	return nil
}

func recoverJS(err *error) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("%v", r)
	}
}

// reorderIMESelection works around Gio's browser text input (see ime.go).
var reorderIMESelection = true
