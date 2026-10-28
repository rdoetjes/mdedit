package ui

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"

	"mdedit/internal/edit"
)

// shortcutCommands returns each command with its own keyboard shortcut once.
func shortcutCommands(menus []*menu) []*command {
	var out []*command
	seen := map[*command]bool{}
	for _, m := range menus {
		for _, c := range m.items {
			if c != nil && c.key != "" && !c.native && !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	return out
}

// handleShortcuts runs commands whose keyboard shortcut was pressed.
func (a *App) handleShortcuts(gtx layout.Context) {
	filters := make([]event.Filter, len(a.shortcuts))
	for i, c := range a.shortcuts {
		filters[i] = key.Filter{Name: c.key, Required: key.ModShortcut | c.mods}
	}
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			return
		}
		if e, isKey := ev.(key.Event); isKey && e.State == key.Press {
			a.runShortcut(gtx, e)
		}
	}
}

func (a *App) runShortcut(gtx layout.Context, e key.Event) {
	for _, c := range a.shortcuts {
		if c.key == e.Name && e.Modifiers == key.ModShortcut|c.mods {
			c.run(gtx)
			return
		}
	}
}

// editorKeys returns the next press of one of the given keys while the
// editor has the focus. Handling them before the editor does keeps the
// editor from acting on them itself.
func (a *App) editorKey(gtx layout.Context, filters ...event.Filter) (key.Event, bool) {
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			return key.Event{}, false
		}
		if e, isKey := ev.(key.Event); isKey && e.State == key.Press {
			return e, true
		}
	}
}

// handleEnter continues lists and quotes when Enter is pressed.
func (a *App) handleEnter(gtx layout.Context) {
	for {
		_, ok := a.editorKey(gtx,
			key.Filter{Focus: &a.editor, Name: key.NameReturn},
			key.Filter{Focus: &a.editor, Name: key.NameEnter})
		if !ok {
			return
		}
		a.applyEdit(gtx, edit.Newline)
	}
}

// handleTab indents with spaces instead of letting Tab move the keyboard
// focus out of the editor.
func (a *App) handleTab(gtx layout.Context) {
	for {
		e, ok := a.editorKey(gtx, key.Filter{Focus: &a.editor, Name: key.NameTab, Optional: key.ModShift})
		if !ok {
			return
		}
		if e.Modifiers.Contain(key.ModShift) {
			a.applyEdit(gtx, edit.Outdent)
		} else {
			a.applyEdit(gtx, edit.Tab)
		}
	}
}

// handleMacKeys adds the standard macOS ⌘ keys that Gio's editor lacks:
// ⌘←/⌘→ to the start/end of the line, ⌘↑/⌘↓ to the start/end of the
// document (with ⇧ to extend the selection), and ⌘⌫ to delete to the start
// of the line.
func (a *App) handleMacKeys(gtx layout.Context) {
	if key.ModShortcut != key.ModCommand {
		return
	}
	filters := a.macKeyFilters()
	for {
		e, ok := a.editorKey(gtx, filters...)
		if !ok {
			return
		}
		if e.Name == key.NameDeleteBackward {
			a.deleteToLineStart()
		} else {
			a.moveCaret(a.macTarget(e.Name), e.Modifiers.Contain(key.ModShift))
		}
	}
}

func (a *App) macKeyFilters() []event.Filter {
	var out []event.Filter
	for _, name := range []key.Name{key.NameLeftArrow, key.NameRightArrow, key.NameUpArrow, key.NameDownArrow} {
		out = append(out, key.Filter{Focus: &a.editor, Name: name, Required: key.ModCommand, Optional: key.ModShift})
	}
	return append(out, key.Filter{Focus: &a.editor, Name: key.NameDeleteBackward, Required: key.ModCommand})
}

// macTarget is where a ⌘-arrow moves the caret.
func (a *App) macTarget(name key.Name) int {
	caret, _ := a.editor.Selection()
	switch name {
	case key.NameLeftArrow:
		return a.visualLineStart(caret)
	case key.NameRightArrow:
		return a.visualLineEnd(caret)
	case key.NameUpArrow:
		return 0
	}
	return a.editor.Len()
}

// moveCaret moves the caret, extending the selection if extend is set.
func (a *App) moveCaret(to int, extend bool) {
	_, anchor := a.editor.Selection()
	if !extend {
		anchor = to
	}
	a.editor.SetCaret(to, anchor)
}

// deleteToLineStart deletes the selection, or from the caret to the start of
// its screen line (joining with the previous line at a line start).
func (a *App) deleteToLineStart() {
	caret, anchor := a.editor.Selection()
	if caret == anchor {
		start := a.visualLineStart(caret)
		if start == caret && caret > 0 {
			start--
		}
		a.editor.SetCaret(start, caret)
	}
	a.editor.Insert("")
	a.reparse()
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
	lo, hi := lineStart([]rune(a.editor.Text()), pos), pos
	if a.visualLines(lo, pos) <= 1 {
		return lo
	}
	// Binary search for the first offset on the same screen line.
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
	lo, hi := pos, lineEnd(text, pos)
	if a.visualLines(pos, hi) <= 1 {
		return hi
	}
	end := hi
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
	if lo > pos && lo < end && text[lo-1] == ' ' {
		lo--
	}
	return lo
}

func lineStart(text []rune, pos int) int {
	for pos > 0 && text[pos-1] != '\n' {
		pos--
	}
	return pos
}

func lineEnd(text []rune, pos int) int {
	for pos < len(text) && text[pos] != '\n' {
		pos++
	}
	return pos
}
