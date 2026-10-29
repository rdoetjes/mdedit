package ui

import (
	"time"
	"unicode/utf8"

	"gioui.org/layout"
	"gioui.org/op"
)

// Typing in a large document doesn't parse and spell-check the whole text
// on every key press: when the last reparse took longer than quickReparse,
// it waits until typing pauses for reparseDelay. Meanwhile the misspelt
// words and outline headings are moved along with the edit, so the wavy
// lines stay under their words. Small documents are still parsed at once.
const (
	quickReparse = 4 * time.Millisecond
	reparseDelay = 150 * time.Millisecond
)

// typedText reacts to a change typed in the editor.
func (a *App) typedText(gtx layout.Context) {
	if a.reparseCost < quickReparse {
		a.reparse()
		return
	}
	txt := a.editor.Text()
	from, oldTo, newTo := changedRange(a.parsedText, txt)
	a.spell.shift(from, oldTo, newTo-oldTo)
	a.outline.shift(from, oldTo, newTo-oldTo)
	a.parsedText = txt
	a.version++ // the find bar searches again
	a.dirty = true
	a.reparseDue = gtx.Now.Add(reparseDelay)
	gtx.Execute(op.InvalidateCmd{At: a.reparseDue})
}

// reparseIfDue runs a postponed reparse once typing has paused.
func (a *App) reparseIfDue(gtx layout.Context) {
	if !a.reparseDue.IsZero() && !gtx.Now.Before(a.reparseDue) {
		a.reparse()
	}
}

// changedRange compares two texts and returns, in runes, where they start
// to differ and where the differing part ends in each: old[from:oldTo]
// became new[from:newTo].
func changedRange(old, new string) (from, oldTo, newTo int) {
	p := 0
	for p < len(old) && p < len(new) && old[p] == new[p] {
		p++
	}
	for p > 0 && (midRune(old, p) || midRune(new, p)) {
		p-- // don't split a character
	}
	q := 0
	for q < len(old)-p && q < len(new)-p && old[len(old)-1-q] == new[len(new)-1-q] {
		q++
	}
	for q > 0 && (midRune(old, len(old)-q) || midRune(new, len(new)-q)) {
		q--
	}
	from = utf8.RuneCountInString(old[:p])
	return from, from + utf8.RuneCountInString(old[p:len(old)-q]), from + utf8.RuneCountInString(new[p:len(new)-q])
}

// midRune reports whether byte i of s is inside a character rather than
// at its start.
func midRune(s string, i int) bool {
	return i < len(s) && !utf8.RuneStart(s[i])
}

// shift moves the misspellings after an edit that replaced [from, to) by
// delta runes, and drops the ones the edit touched; they are checked again
// at the next reparse.
func (s *spellChecker) shift(from, to, delta int) {
	kept := s.misses[:0]
	for _, m := range s.misses {
		switch {
		case m.End < from:
			kept = append(kept, m)
		case m.Start > to:
			m.Start, m.End = m.Start+delta, m.End+delta
			kept = append(kept, m)
		}
	}
	s.misses = kept
}

// shift moves the headings after an edit that replaced [from, to) by
// delta runes.
func (o *outline) shift(from, to, delta int) {
	for i := range o.items {
		switch {
		case o.items[i].offset >= to:
			o.items[i].offset += delta
		case o.items[i].offset > from:
			o.items[i].offset = from
		}
	}
}
