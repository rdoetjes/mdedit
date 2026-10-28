package ui

import (
	"io"
	"strings"

	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/layout"

	"mdedit/internal/edit"
)

// editFunc is a formatting operation on the editor text and selection.
type editFunc func(edit.State) edit.State

// editCommand makes a command that applies f to the editor.
func (a *App) editCommand(label string, k key.Name, mods key.Modifiers, f editFunc) *command {
	return newCommand(label, k, mods, editAction{a: a, f: f}.run)
}

// editAction applies an edit when run.
type editAction struct {
	a *App
	f editFunc
}

func (e editAction) run(gtx layout.Context) { e.a.applyEdit(gtx, e.f) }

// Parameterised edits, as types whose apply method is an editFunc.

type inlineMarker [2]string

func (m inlineMarker) apply(s edit.State) edit.State { return edit.ToggleInline(s, m) }

type alignment string

func (al alignment) apply(s edit.State) edit.State { return edit.SetAlign(s, string(al)) }

type listStyle edit.ListKind

func (l listStyle) apply(s edit.State) edit.State { return edit.ToggleList(s, edit.ListKind(l)) }

type headingLevel int

func (h headingLevel) apply(s edit.State) edit.State { return edit.SetHeading(s, int(h)) }

type linkKind bool // true for images

func (k linkKind) apply(s edit.State) edit.State { return edit.Link(s, bool(k)) }

type tableSize struct{ cols, rows int }

func (t tableSize) apply(s edit.State) edit.State { return edit.Table(s, t.cols, t.rows) }

// applyEdit runs a formatting function on the editor contents. The change is
// applied as a single replacement of the part that differs, so it becomes
// one undo step.
func (a *App) applyEdit(gtx layout.Context, f editFunc) {
	s, e := a.editor.Selection()
	old := []rune(a.editor.Text())
	ns := f(edit.New(string(old), s, e))
	p, q := commonPrefix(old, ns.Text), commonSuffix(old, ns.Text)
	if p != len(old) || p != len(ns.Text) {
		q = min(q, len(old)-p, len(ns.Text)-p)
		a.editor.SetCaret(p, len(old)-q)
		a.editor.Insert(string(ns.Text[p : len(ns.Text)-q]))
		a.reparse()
	}
	a.editor.SetCaret(ns.End, ns.Start)
	gtx.Execute(key.FocusCmd{Tag: &a.editor})
}

func commonPrefix(a, b []rune) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}

func commonSuffix(a, b []rune) int {
	n := 0
	for n < len(a) && n < len(b) && a[len(a)-1-n] == b[len(b)-1-n] {
		n++
	}
	return n
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
