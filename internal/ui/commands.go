package ui

import (
	"io"
	"strings"

	"gioui.org/io/clipboard"
	"gioui.org/io/event"
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

func (c *command) shortcut() string {
	if c.key == "" {
		return ""
	}
	mac := key.ModShortcut == key.ModCommand
	var parts []string
	if mac {
		if c.mods.Contain(key.ModAlt) {
			parts = append(parts, "⌥")
		}
		if c.mods.Contain(key.ModShift) {
			parts = append(parts, "⇧")
		}
		return strings.Join(parts, "") + "⌘" + string(c.key)
	}
	parts = append(parts, "Ctrl")
	if c.mods.Contain(key.ModAlt) {
		parts = append(parts, "Alt")
	}
	if c.mods.Contain(key.ModShift) {
		parts = append(parts, "Shift")
	}
	return strings.Join(append(parts, string(c.key)), "+")
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

func mustIcon(data []byte) *widget.Icon {
	ic, err := widget.NewIcon(data)
	if err != nil {
		panic(err)
	}
	return ic
}

// buildCommands sets up the menus and the toolbar.
func (a *App) buildCommands() {
	sc := func(label string, k key.Name, mods key.Modifiers, run func(gtx layout.Context)) *command {
		return &command{label: label, key: k, mods: mods, run: run}
	}
	plain := func(label string, run func()) *command {
		return &command{label: label, run: func(layout.Context) { run() }}
	}
	format := func(label string, k key.Name, mods key.Modifiers, f func(edit.State) edit.State) *command {
		return sc(label, k, mods, func(gtx layout.Context) { a.applyEdit(gtx, f) })
	}
	inline := func(m [2]string) func(edit.State) edit.State {
		return func(s edit.State) edit.State { return edit.ToggleInline(s, m) }
	}
	align := func(al string) func(edit.State) edit.State {
		return func(s edit.State) edit.State { return edit.SetAlign(s, al) }
	}
	list := func(k edit.ListKind) func(edit.State) edit.State {
		return func(s edit.State) edit.State { return edit.ToggleList(s, k) }
	}
	heading := func(n int) func(edit.State) edit.State {
		return func(s edit.State) edit.State { return edit.SetHeading(s, n) }
	}
	const shift = key.ModShift

	bold := format("Bold", "B", 0, inline(edit.Bold))
	italic := format("Italic", "I", 0, inline(edit.Italic))
	underline := format("Underline", "U", 0, inline(edit.Underline))
	left := format("Align Left", "L", shift, align(edit.AlignLeft))
	center := format("Align Center", "E", shift, align(edit.AlignCenter))
	right := format("Align Right", "R", shift, align(edit.AlignRight))
	numbered := format("Numbered List", "", 0, list(edit.Numbers))
	bullets := format("Bullet List", "", 0, list(edit.Bullets))
	table := sc("Table…", "T", 0, func(layout.Context) { a.tableDialog() })

	bold.icon = mustIcon(icons.EditorFormatBold)
	italic.icon = mustIcon(icons.EditorFormatItalic)
	underline.icon = mustIcon(icons.EditorFormatUnderlined)
	left.icon = mustIcon(icons.EditorFormatAlignLeft)
	center.icon = mustIcon(icons.EditorFormatAlignCenter)
	right.icon = mustIcon(icons.EditorFormatAlignRight)
	numbered.icon = mustIcon(icons.EditorFormatListNumbered)
	bullets.icon = mustIcon(icons.EditorFormatListBulleted)
	table.icon = mustIcon(icons.EditorBorderAll)
	// Toolbar groups; nil separates them.
	a.toolbarCmds = []*command{bold, italic, underline, nil, left, center, right, nil, numbered, bullets, table}

	a.insertTOC = format("Table of Contents", "", 0, edit.InsertTOC)
	a.updateTOC = sc("Update Table of Contents", "", 0, func(gtx layout.Context) {
		if _, _, ok := edit.FindTOC(edit.New(a.editor.Text(), 0, 0)); !ok {
			a.setStatus("This document has no table of contents; use Insert → Table of Contents")
			return
		}
		a.applyEdit(gtx, edit.UpdateTOC)
		a.setStatus("Table of contents updated")
	})
	a.cutCmd = &command{label: "Cut", key: "X", native: true, run: a.cut}
	a.copyCmd = &command{label: "Copy", key: "C", native: true, run: a.copy}
	a.pasteCmd = &command{label: "Paste", key: "V", native: true, run: func(gtx layout.Context) {
		gtx.Execute(clipboard.ReadCmd{Tag: &a.editor})
		gtx.Execute(key.FocusCmd{Tag: &a.editor})
	}}

	view := func(label string, v viewMode) *command {
		return plain(label, func() { a.view = v })
	}

	a.menus = []*menu{
		{"File", []*command{
			sc("New", "N", 0, func(layout.Context) { a.newDoc() }),
			sc("Open…", "O", 0, func(layout.Context) { a.openDialog() }),
			nil,
			sc("Save", "S", 0, func(layout.Context) { a.save(nil) }),
			sc("Save As…", "S", shift, func(layout.Context) { a.saveAs(nil) }),
			nil,
			sc("Export PDF…", "E", 0, func(layout.Context) { a.exportDialog() }),
			nil,
			sc("Quit", "Q", 0, func(layout.Context) { a.quit() }),
		}},
		{"Edit", []*command{
			a.cutCmd, a.copyCmd, a.pasteCmd,
			nil,
			{label: "Select All", key: "A", native: true, run: func(gtx layout.Context) {
				a.editor.SetCaret(a.editor.Len(), 0)
				gtx.Execute(key.FocusCmd{Tag: &a.editor})
			}},
		}},
		{"Format", []*command{
			bold, italic, underline,
			format("Strikethrough", "X", shift, inline(edit.Strike)),
			format("Inline Code", "", 0, inline(edit.Code)),
			format("Inline Math", "", 0, inline(edit.InlineMath)),
			nil,
			left, center, right,
			nil,
			format("Heading 1", "1", 0, heading(1)),
			format("Heading 2", "2", 0, heading(2)),
			format("Heading 3", "3", 0, heading(3)),
			format("Heading 4", "4", 0, heading(4)),
			format("Heading 5", "5", 0, heading(5)),
			format("Heading 6", "6", 0, heading(6)),
			format("Paragraph", "0", 0, heading(0)),
		}},
		{"Insert", []*command{
			numbered, bullets,
			format("Task List", "", 0, list(edit.Tasks)),
			format("Quote", "Q", shift, edit.ToggleQuote),
			nil,
			table,
			format("Code Block", "K", shift, edit.CodeBlock),
			format("Math Block", "M", shift, edit.MathBlock),
			format("Horizontal Rule", "", 0, edit.Rule),
			nil,
			a.insertTOC, a.updateTOC,
			nil,
			format("Link", "K", 0, func(s edit.State) edit.State { return edit.Link(s, false) }),
			format("Image", "I", shift, func(s edit.State) edit.State { return edit.Link(s, true) }),
		}},
		{"View", []*command{
			view("Editor and Preview", viewSplit),
			view("Editor Only", viewEditor),
			view("Preview Only", viewPreview),
			nil,
			sc("Cycle Views", "/", 0, func(layout.Context) { a.view = (a.view + 1) % 3 }),
		}},
		{"Help", []*command{
			plain("Markdown Reference", func() { openURL("https://commonmark.org/help/") }),
			plain("About mdedit", a.about),
		}},
	}
}

// contextItems returns the right-click menu: clipboard commands, then
// insert or update the table of contents.
func (a *App) contextItems() []*command {
	toc := a.insertTOC
	if _, _, ok := edit.FindTOC(edit.New(a.editor.Text(), 0, 0)); ok {
		toc = a.updateTOC
	}
	return []*command{a.cutCmd, a.copyCmd, a.pasteCmd, nil, toc}
}

// handleShortcuts runs commands whose keyboard shortcut was pressed.
func (a *App) handleShortcuts(gtx layout.Context) {
	var filters []event.Filter
	var cmds []*command
	seen := map[*command]bool{}
	for _, m := range a.menus {
		for _, c := range m.items {
			if c == nil || c.key == "" || c.native || seen[c] {
				continue
			}
			seen[c] = true
			cmds = append(cmds, c)
			filters = append(filters, key.Filter{Name: c.key, Required: key.ModShortcut | c.mods})
		}
	}
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		for _, c := range cmds {
			if c.key == e.Name && e.Modifiers == key.ModShortcut|c.mods {
				c.run(gtx)
				break
			}
		}
	}
}

// handleEnter continues lists and quotes when Enter is pressed in the
// editor. It must run before the editor sees the key.
func (a *App) handleEnter(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(
			key.Filter{Focus: &a.editor, Name: key.NameReturn},
			key.Filter{Focus: &a.editor, Name: key.NameEnter},
		)
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			a.applyEdit(gtx, edit.Newline)
		}
	}
}

// handleTab indents with spaces instead of letting Tab move the keyboard
// focus out of the editor.
func (a *App) handleTab(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(key.Filter{Focus: &a.editor, Name: key.NameTab, Optional: key.ModShift})
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		if e.Modifiers.Contain(key.ModShift) {
			a.applyEdit(gtx, edit.Outdent)
		} else {
			a.applyEdit(gtx, edit.Tab)
		}
	}
}

// handleMacKeys adds the standard macOS ⌘ navigation keys that Gio's editor
// lacks: ⌘←/⌘→ to the start/end of the line, ⌘↑/⌘↓ to the start/end of the
// document (with ⇧ to extend the selection), and ⌘⌫ to delete to the start
// of the line. Must run before the editor sees the keys.
func (a *App) handleMacKeys(gtx layout.Context) {
	if key.ModShortcut != key.ModCommand {
		return
	}
	ed := &a.editor
	nav := func(name key.Name) key.Filter {
		return key.Filter{Focus: ed, Name: name, Required: key.ModCommand, Optional: key.ModShift}
	}
	for {
		ev, ok := gtx.Event(
			nav(key.NameLeftArrow), nav(key.NameRightArrow),
			nav(key.NameUpArrow), nav(key.NameDownArrow),
			key.Filter{Focus: ed, Name: key.NameDeleteBackward, Required: key.ModCommand},
		)
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		caret, anchor := ed.Selection()
		var target int
		switch e.Name {
		case key.NameLeftArrow:
			target = a.visualLineStart(caret)
		case key.NameRightArrow:
			target = a.visualLineEnd(caret)
		case key.NameUpArrow:
			target = 0
		case key.NameDownArrow:
			target = ed.Len()
		case key.NameDeleteBackward:
			if caret == anchor {
				start := a.visualLineStart(caret)
				if start == caret && caret > 0 {
					start-- // at a line start: join with the previous line
				}
				ed.SetCaret(start, caret)
			}
			ed.Insert("")
			a.reparse()
			continue
		}
		if e.Modifiers.Contain(key.ModShift) {
			ed.SetCaret(target, anchor)
		} else {
			ed.SetCaret(target, target)
		}
	}
}

// visualLines counts the screen lines the text between p and q is shown on.
// It returns 0 if that text is scrolled out of view.
func (a *App) visualLines(p, q int) int {
	if p == q {
		return 1
	}
	a.regions = a.editor.Regions(p, q, a.regions[:0])
	return len(a.regions)
}

// visualLineStart returns the offset where the wrapped screen line holding
// pos begins.
func (a *App) visualLineStart(pos int) int {
	text := []rune(a.editor.Text())
	ls := pos
	for ls > 0 && text[ls-1] != '\n' {
		ls--
	}
	if a.visualLines(ls, pos) <= 1 {
		return ls
	}
	lo, hi := ls, pos
	for lo < hi {
		mid := (lo + hi) / 2
		if a.visualLines(mid, pos) <= 1 {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return lo
}

// visualLineEnd returns the offset where the wrapped screen line holding pos
// ends.
func (a *App) visualLineEnd(pos int) int {
	text := []rune(a.editor.Text())
	le := pos
	for le < len(text) && text[le] != '\n' {
		le++
	}
	if a.visualLines(pos, le) <= 1 {
		return le
	}
	lo, hi := pos, le
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if a.visualLines(pos, mid) <= 1 {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	// Stop before the space where the line wrapped, so the caret stays on
	// this screen line.
	if lo > pos && lo < le && text[lo-1] == ' ' {
		lo--
	}
	return lo
}

// applyEdit runs a formatting function on the editor contents. The change is
// applied as a single replacement of the part that differs, so it becomes
// one undo step.
func (a *App) applyEdit(gtx layout.Context, f func(edit.State) edit.State) {
	s, e := a.editor.Selection()
	old := []rune(a.editor.Text())
	ns := f(edit.New(string(old), s, e))
	nw := ns.Text

	p := 0
	for p < len(old) && p < len(nw) && old[p] == nw[p] {
		p++
	}
	q := 0
	for q < len(old)-p && q < len(nw)-p && old[len(old)-1-q] == nw[len(nw)-1-q] {
		q++
	}
	if p != len(old) || p != len(nw) {
		a.editor.SetCaret(p, len(old)-q)
		a.editor.Insert(string(nw[p : len(nw)-q]))
		a.reparse()
	}
	a.editor.SetCaret(ns.End, ns.Start)
	gtx.Execute(key.FocusCmd{Tag: &a.editor})
}

func (a *App) copy(gtx layout.Context) {
	if sel := a.editor.SelectedText(); sel != "" {
		gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(sel))})
	}
	gtx.Execute(key.FocusCmd{Tag: &a.editor})
}

func (a *App) cut(gtx layout.Context) {
	if a.editor.SelectedText() == "" {
		return
	}
	a.copy(gtx)
	a.editor.Insert("")
	a.reparse()
}

func (a *App) tableDialog() {
	a.dialog = NewTableDialog(func(cols, rows int) {
		a.dialog = nil
		if cols > 0 {
			a.pending = func(gtx layout.Context) {
				a.applyEdit(gtx, func(s edit.State) edit.State { return edit.Table(s, cols, rows) })
			}
		}
	})
}

func (a *App) about() {
	a.dialog = &ConfirmDialog{
		Message: "mdedit " + Version + "\n\nA Markdown editor with live preview and PDF export, written in Go with Gio.",
		Buttons: []string{"OK"},
		Done:    func(int) { a.dialog = nil },
	}
}
