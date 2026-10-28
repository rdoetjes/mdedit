package ui

import (
	"strings"

	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/widget"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"mdedit/internal/edit"
)

// command is an action reachable from the menu bar, and optionally from the
// toolbar and a keyboard shortcut.
type command struct {
	label string
	// Shortcut: key plus modifiers on top of the platform shortcut modifier
	// (Ctrl, or ⌘ on macOS).
	key  key.Name
	mods key.Modifiers
	// native shortcuts are handled by the text editor itself; they are only
	// shown in the menu.
	native bool
	run    func(gtx layout.Context)

	icon  *widget.Icon // toolbar icon, if the command is on the toolbar
	click widget.Clickable
}

func newCommand(label string, k key.Name, mods key.Modifiers, run func(gtx layout.Context)) *command {
	return &command{label: label, key: k, mods: mods, run: run}
}

// shortcut describes the keyboard shortcut in the platform's notation.
func (c *command) shortcut() string {
	if c.key == "" {
		return ""
	}
	if key.ModShortcut == key.ModCommand {
		return c.macShortcut()
	}
	parts := []string{"Ctrl"}
	if c.mods.Contain(key.ModAlt) {
		parts = append(parts, "Alt")
	}
	if c.mods.Contain(key.ModShift) {
		parts = append(parts, "Shift")
	}
	return strings.Join(append(parts, string(c.key)), "+")
}

func (c *command) macShortcut() string {
	s := ""
	if c.mods.Contain(key.ModAlt) {
		s += "⌥"
	}
	if c.mods.Contain(key.ModShift) {
		s += "⇧"
	}
	return s + "⌘" + string(c.key)
}

// tooltip is shown in the status bar while hovering a toolbar button.
func (c *command) tooltip() string {
	if s := c.shortcut(); s != "" {
		return c.label + "  (" + s + ")"
	}
	return c.label
}

type menu struct {
	title string
	items []*command // nil entries are separators
}

// commands are the commands referred to from more than one place.
type commands struct {
	bold, italic, underline  *command
	left, center, right      *command
	numbered, bullets, table *command
	cut, copy, paste         *command
	insertTOC, updateTOC     *command
	find, replace            *command
	video                    *command
	theme                    *command
}

func mustIcon(data []byte) *widget.Icon {
	ic, err := widget.NewIcon(data)
	if err != nil {
		panic(err)
	}
	return ic
}

// buildCommands sets up the commands, the menus, the toolbar and the
// keyboard shortcuts.
func (a *App) buildCommands() {
	a.buildSharedCommands()
	a.buildToolbar()
	a.menus = []*menu{a.fileMenu(), a.editMenu(), a.formatMenu(), a.insertMenu(), a.viewMenu(), a.helpMenu()}
	a.shortcuts = shortcutCommands(a.menus)
}

func (a *App) buildSharedCommands() {
	c := &a.cmds
	c.bold = a.editCommand("Bold", "B", 0, inlineMarker(edit.Bold).apply)
	c.italic = a.editCommand("Italic", "I", 0, inlineMarker(edit.Italic).apply)
	c.underline = a.editCommand("Underline", "U", 0, inlineMarker(edit.Underline).apply)
	c.left = a.editCommand("Align Left", "L", key.ModShift, alignment(edit.AlignLeft).apply)
	c.center = a.editCommand("Align Center", "E", key.ModShift, alignment(edit.AlignCenter).apply)
	c.right = a.editCommand("Align Right", "R", key.ModShift, alignment(edit.AlignRight).apply)
	c.numbered = a.editCommand("Numbered List", "", 0, listStyle(edit.Numbers).apply)
	c.bullets = a.editCommand("Bullet List", "", 0, listStyle(edit.Bullets).apply)
	c.table = newCommand("Table…", "T", 0, a.cmdTable)
	c.cut = &command{label: "Cut", key: "X", native: true, run: a.cut}
	c.copy = &command{label: "Copy", key: "C", native: true, run: a.copy}
	c.paste = &command{label: "Paste", key: "V", native: true, run: a.paste}
	c.insertTOC = a.editCommand("Table of Contents", "", 0, edit.InsertTOC)
	c.updateTOC = newCommand("Update Table of Contents", "", 0, a.cmdUpdateTOC)
	c.video = newCommand("YouTube Video…", "", 0, a.cmdVideo)
	c.theme = newCommand("Dark Theme", "D", key.ModShift, a.cmdToggleTheme)
	c.find = newCommand("Find…", "F", 0, a.cmdFind)
	c.replace = newCommand("Replace…", "R", 0, a.cmdReplace)
}

func (a *App) buildToolbar() {
	c := &a.cmds
	c.bold.icon = mustIcon(icons.EditorFormatBold)
	c.italic.icon = mustIcon(icons.EditorFormatItalic)
	c.underline.icon = mustIcon(icons.EditorFormatUnderlined)
	c.left.icon = mustIcon(icons.EditorFormatAlignLeft)
	c.center.icon = mustIcon(icons.EditorFormatAlignCenter)
	c.right.icon = mustIcon(icons.EditorFormatAlignRight)
	c.numbered.icon = mustIcon(icons.EditorFormatListNumbered)
	c.bullets.icon = mustIcon(icons.EditorFormatListBulleted)
	c.table.icon = mustIcon(icons.EditorBorderAll)
	c.video.icon = mustIcon(icons.AVVideoLibrary)
	c.find.icon = mustIcon(icons.ActionSearch)
	c.replace.icon = mustIcon(icons.ActionFindReplace)
	// Groups are separated by nil.
	a.toolbarCmds = []*command{c.bold, c.italic, c.underline, nil, c.left, c.center, c.right,
		nil, c.numbered, c.bullets, c.table, c.video, nil, c.find, c.replace}
}

func (a *App) fileMenu() *menu {
	items := []*command{
		newCommand("New", "N", 0, a.cmdNew),
		newCommand("Open…", "O", 0, a.cmdOpen),
		nil,
		newCommand("Save", "S", 0, a.cmdSave),
		newCommand("Save As…", "S", key.ModShift, a.cmdSaveAs),
		nil,
		newCommand("Export PDF…", "E", 0, a.cmdExport),
	}
	return &menu{"File", append(items, a.quitCommands()...)}
}

func (a *App) editMenu() *menu {
	return &menu{"Edit", []*command{
		a.cmds.cut, a.cmds.copy, a.cmds.paste,
		nil,
		{label: "Select All", key: "A", native: true, run: a.selectAll},
		nil,
		a.cmds.find, a.cmds.replace,
		newCommand("Find Next", "G", 0, a.cmdFindNext),
		newCommand("Find Previous", "G", key.ModShift, a.cmdFindPrev),
	}}
}

func (a *App) formatMenu() *menu {
	c := &a.cmds
	return &menu{"Format", []*command{
		c.bold, c.italic, c.underline,
		a.editCommand("Strikethrough", "X", key.ModShift, inlineMarker(edit.Strike).apply),
		a.editCommand("Inline Code", "", 0, inlineMarker(edit.Code).apply),
		a.editCommand("Inline Math", "", 0, inlineMarker(edit.InlineMath).apply),
		nil,
		c.left, c.center, c.right,
		nil,
		a.editCommand("Heading 1", "1", 0, headingLevel(1).apply),
		a.editCommand("Heading 2", "2", 0, headingLevel(2).apply),
		a.editCommand("Heading 3", "3", 0, headingLevel(3).apply),
		a.editCommand("Heading 4", "4", 0, headingLevel(4).apply),
		a.editCommand("Heading 5", "5", 0, headingLevel(5).apply),
		a.editCommand("Heading 6", "6", 0, headingLevel(6).apply),
		a.editCommand("Paragraph", "0", 0, headingLevel(0).apply),
	}}
}

func (a *App) insertMenu() *menu {
	c := &a.cmds
	return &menu{"Insert", []*command{
		c.numbered, c.bullets,
		a.editCommand("Task List", "", 0, listStyle(edit.Tasks).apply),
		a.editCommand("Quote", "Q", key.ModShift, edit.ToggleQuote),
		nil,
		c.table,
		a.editCommand("Code Block", "K", key.ModShift, edit.CodeBlock),
		a.editCommand("Math Block", "M", key.ModShift, edit.MathBlock),
		a.editCommand("Horizontal Rule", "", 0, edit.Rule),
		nil,
		c.insertTOC, c.updateTOC,
		nil,
		a.editCommand("Link", "K", 0, linkKind(false).apply),
		a.editCommand("Image", "I", key.ModShift, linkKind(true).apply),
		c.video,
	}}
}

func (a *App) viewMenu() *menu {
	return &menu{"View", []*command{
		newCommand("Editor and Preview", "", 0, viewChoice{a, viewSplit}.run),
		newCommand("Editor Only", "", 0, viewChoice{a, viewEditor}.run),
		newCommand("Preview Only", "", 0, viewChoice{a, viewPreview}.run),
		nil,
		a.cmds.theme,
		nil,
		newCommand("Cycle Views", "/", 0, a.cmdCycleView),
	}}
}

func (a *App) helpMenu() *menu {
	return &menu{"Help", []*command{
		newCommand("Markdown Reference", "", 0, a.cmdMarkdownHelp),
		newCommand("About mdedit", "", 0, a.cmdAbout),
	}}
}

// contextItems returns the right-click menu: clipboard commands, then
// insert or update the table of contents.
func (a *App) contextItems() []*command {
	toc := a.cmds.insertTOC
	if a.hasTOC() {
		toc = a.cmds.updateTOC
	}
	return []*command{a.cmds.cut, a.cmds.copy, a.cmds.paste, nil, toc}
}

func (a *App) hasTOC() bool {
	_, _, ok := edit.FindTOC(edit.New(a.editor.Text(), 0, 0))
	return ok
}

// ---- Command actions ----

func (a *App) cmdNew(layout.Context)    { a.newDoc() }
func (a *App) cmdOpen(layout.Context)   { a.openDialog() }
func (a *App) cmdSave(layout.Context)   { a.save(actNone) }
func (a *App) cmdSaveAs(layout.Context) { a.saveAs(actNone) }
func (a *App) cmdExport(layout.Context) { a.exportDialog() }
func (a *App) cmdQuit(layout.Context)   { a.quit() }
func (a *App) cmdTable(layout.Context)  { a.tableDialog() }
func (a *App) cmdAbout(layout.Context)  { a.about() }

func (a *App) cmdCycleView(layout.Context) { a.view = (a.view + 1) % 3 }

func (a *App) cmdFind(gtx layout.Context)    { a.find.show(gtx, false) }
func (a *App) cmdReplace(gtx layout.Context) { a.find.show(gtx, true) }

// cmdFindNext goes to the next match, opening the bar if it is closed.
func (a *App) cmdFindNext(gtx layout.Context) {
	if !a.find.open {
		a.find.show(gtx, false)
	}
	a.find.findNext()
}

func (a *App) cmdFindPrev(gtx layout.Context) {
	if !a.find.open {
		a.find.show(gtx, false)
	}
	a.find.findPrev()
}

func (a *App) cmdMarkdownHelp(layout.Context) { a.openExternal("https://commonmark.org/help/") }

func (a *App) cmdUpdateTOC(gtx layout.Context) {
	if !a.hasTOC() {
		a.setStatus("This document has no table of contents; use Insert → Table of Contents")
		return
	}
	a.applyEdit(gtx, edit.UpdateTOC)
	a.setStatus("Table of contents updated")
}

func (a *App) selectAll(gtx layout.Context) {
	a.editor.SetCaret(a.editor.Len(), 0)
	gtx.Execute(key.FocusCmd{Tag: &a.editor})
}

func (a *App) paste(gtx layout.Context) {
	gtx.Execute(clipboard.ReadCmd{Tag: &a.editor})
	gtx.Execute(key.FocusCmd{Tag: &a.editor})
}

func (a *App) cmdVideo(layout.Context) { a.dialog = NewVideoDialog(a.videoChosen) }

func (a *App) videoChosen(id, title string) {
	a.dialog = nil
	if id != "" {
		a.pendingEdit = videoLink{id, title}.apply
	}
}

func (a *App) tableDialog() {
	a.dialog = NewTableDialog(a.tableChosen)
}

func (a *App) tableChosen(cols, rows int) {
	a.dialog = nil
	if cols > 0 {
		a.pendingEdit = tableSize{cols, rows}.apply
	}
}

func (a *App) about() {
	a.dialog = &ConfirmDialog{
		Message: "mdedit " + Version + "\n\nA Markdown editor with live preview and PDF export, written in Go with Gio.",
		Buttons: []string{"OK"},
		Done:    a.closeDialog,
	}
}

func (a *App) closeDialog(int) { a.dialog = nil }

// viewChoice switches to a view mode.
type viewChoice struct {
	a *App
	v viewMode
}

func (c viewChoice) run(layout.Context) { c.a.view = c.v }
