package ui

import (
	"fmt"
	"strings"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"mdedit/internal/edit"
)

// maxMatches limits how many matches are found and highlighted.
const maxMatches = 10000

// findBar is the search and replace bar above the editor.
type findBar struct {
	a         *App
	th        *material.Theme
	open      bool
	replace   bool // show the replace row
	query     widget.Editor
	repl      widget.Editor
	matchCase widget.Bool
	prev      widget.Clickable
	next      widget.Clickable
	close     widget.Clickable
	replOne   widget.Clickable
	replAll   widget.Clickable
	icons     findIcons

	matches []int // rune offsets of the matches
	current int   // index of the selected match, or -1
	key     searchKey
}

type findIcons struct {
	search, up, down, close *widget.Icon
}

// searchKey identifies what the matches were computed for.
type searchKey struct {
	q       edit.Query
	version int
}

func (f *findBar) init(a *App) {
	f.a = a
	f.th = a.th
	f.current = -1
	for _, e := range []*widget.Editor{&f.query, &f.repl} {
		e.SingleLine = true
		e.Submit = true
	}
	f.icons = findIcons{
		search: mustIcon(icons.ActionSearch),
		up:     mustIcon(icons.HardwareKeyboardArrowUp),
		down:   mustIcon(icons.HardwareKeyboardArrowDown),
		close:  mustIcon(icons.NavigationClose),
	}
}

// show opens the bar, with the replace row if withReplace is set. The
// selected text, if it's on one line, becomes the search text.
func (f *findBar) show(gtx layout.Context, withReplace bool) {
	if !f.open {
		f.replace = false
	}
	f.open = true
	f.replace = f.replace || withReplace
	if sel := f.a.editor.SelectedText(); sel != "" && !strings.ContainsRune(sel, '\n') {
		f.query.SetText(sel)
	}
	f.query.SetCaret(f.query.Len(), 0) // select all, so typing replaces it
	if f.a.view == viewPreview {
		f.a.view = viewSplit
	}
	gtx.Execute(key.FocusCmd{Tag: &f.query})
}

func (f *findBar) hide(gtx layout.Context) {
	f.open = false
	f.replace = false
	gtx.Execute(key.FocusCmd{Tag: &f.a.editor})
}

func (f *findBar) currentQuery() edit.Query {
	return edit.Query{Text: f.query.Text(), MatchCase: f.matchCase.Value}
}

// update searches again if the query or the text changed.
func (f *findBar) update() {
	k := searchKey{q: f.currentQuery(), version: f.a.version}
	if k == f.key {
		return
	}
	f.key = k
	f.matches = edit.FindAll([]rune(f.a.editor.Text()), k.q, maxMatches)
	f.current = f.selectedMatch()
}

// selectedMatch is the index of the match the editor selection is on, or -1.
func (f *findBar) selectedMatch() int {
	s := f.a.editorState()
	if !f.key.q.Matches(s) {
		return -1
	}
	for i, m := range f.matches {
		if m == s.Start {
			return i
		}
	}
	return -1
}

// findNext selects the next match after the selection, wrapping around.
func (f *findBar) findNext() {
	f.update()
	f.selectMatch(edit.NextMatch(f.matches, f.a.editorState().End))
}

// findPrev selects the match before the selection, wrapping around.
func (f *findBar) findPrev() {
	f.update()
	f.selectMatch(edit.PrevMatch(f.matches, f.a.editorState().Start))
}

func (f *findBar) selectMatch(i int) {
	f.current = i
	if i < 0 {
		return
	}
	m := f.matches[i]
	f.a.editor.SetCaret(m+len([]rune(f.key.q.Text)), m)
}

// replaceOne replaces the selected match, then selects the next one.
func (f *findBar) replaceOne() {
	f.update()
	if ns, ok := edit.ReplaceSelection(f.a.editorState(), f.key.q, f.repl.Text()); ok {
		f.a.setEditorState(ns)
		f.update()
	}
	f.findNext()
}

func (f *findBar) replaceAll() {
	f.update()
	ns, n := edit.ReplaceAll(f.a.editorState(), f.key.q, f.repl.Text())
	if n > 0 {
		f.a.setEditorState(ns)
	}
	f.a.setStatus("Replaced %d occurrence%s", n, plural(n))
	f.update()
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// counter describes the matches: "3 of 12", "12 matches" or "No results".
func (f *findBar) counter() string {
	switch {
	case f.key.q.Text == "":
		return ""
	case len(f.matches) == 0:
		return "No results"
	case f.current >= 0:
		return fmt.Sprintf("%d of %d", f.current+1, len(f.matches))
	}
	if len(f.matches) == 1 {
		return "1 match"
	}
	return fmt.Sprintf("%d matches", len(f.matches))
}

// handleInput processes the bar's keys and buttons.
func (f *findBar) handleInput(gtx layout.Context) {
	if !f.open {
		return
	}
	if f.closeRequested(gtx) {
		f.hide(gtx)
		return
	}
	if f.shiftEnter(gtx) || f.prev.Clicked(gtx) {
		f.findPrev()
	}
	if f.submitted(gtx, &f.query) || f.next.Clicked(gtx) {
		f.findNext()
	}
	if f.submitted(gtx, &f.repl) || f.replOne.Clicked(gtx) {
		f.replaceOne()
	}
	if f.replAll.Clicked(gtx) {
		f.replaceAll()
	}
	f.matchCase.Update(gtx)
}

// closeRequested reports Escape in one of the bar's fields, or the close
// button.
func (f *findBar) closeRequested(gtx layout.Context) bool {
	_, esc := keyPressedIn(gtx, &f.query, key.NameEscape, 0)
	_, escRepl := keyPressedIn(gtx, &f.repl, key.NameEscape, 0)
	return esc || escRepl || f.close.Clicked(gtx)
}

// shiftEnter reports Shift+Enter in the search field, which goes back.
func (f *findBar) shiftEnter(gtx layout.Context) bool {
	_, ok := keyPressedIn(gtx, &f.query, key.NameReturn, key.ModShift)
	return ok
}

// submitted reports Enter in a field.
func (f *findBar) submitted(gtx layout.Context, e *widget.Editor) bool {
	sub := false
	for {
		ev, ok := e.Update(gtx)
		if !ok {
			return sub
		}
		if _, isSubmit := ev.(widget.SubmitEvent); isSubmit {
			sub = true
		}
	}
}

// keyPressedIn drains presses of a key with exactly the given modifiers
// while tag has the focus, and reports whether there was one.
func keyPressedIn(gtx layout.Context, tag *widget.Editor, name key.Name, mods key.Modifiers) (key.Event, bool) {
	var found key.Event
	ok := false
	for {
		ev, more := gtx.Event(key.Filter{Focus: tag, Name: name, Required: mods})
		if !more {
			return found, ok
		}
		if e, isKey := ev.(key.Event); isKey && e.State == key.Press {
			found, ok = e, true
		}
	}
}
