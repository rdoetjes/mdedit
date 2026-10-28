package ui

import (
	"image"
	"slices"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"mdedit/internal/edit"
	"mdedit/internal/mdmodel"
	"mdedit/internal/spell"
)

// maxSuggestions is the number of corrections offered in the context menu.
const maxSuggestions = 5

// spellChecker underlines misspelt words in the editor and offers
// corrections for them in the context menu.
type spellChecker struct {
	a      *App
	dict   *spell.Dictionary
	on     bool
	misses []spell.Miss
	// tag receives right-clicks on the editor text, in editor coordinates.
	tag     int
	boxes   []missBox // where the misses were underlined in the last frame
	regions []widget.Region
	// clicked is the miss right-clicked in this frame, if any.
	clicked *spell.Miss
}

// missBox is the area of (one line of) a misspelt word in the editor.
type missBox struct {
	r image.Rectangle
	i int // index into misses
}

// init loads the dictionary and the words the user added to it.
func (s *spellChecker) init(a *App, st settings) {
	s.a, s.dict = a, spell.English()
	for _, w := range st.Words {
		s.dict.Add(w)
	}
	s.setOn(!st.NoSpelling)
}

func (s *spellChecker) setOn(on bool) {
	s.on = on
	s.a.cmds.spelling.label = "Show Spelling Errors"
	if on {
		s.a.cmds.spelling.label = "Hide Spelling Errors"
	}
}

// check finds the misspelt words of the editor text.
func (s *spellChecker) check(text string, doc *mdmodel.Document) {
	s.misses = nil
	if s.on {
		s.misses = spell.Check(s.dict, []rune(text), doc)
	}
}

// recheck checks the editor text again, after the dictionary changed.
func (s *spellChecker) recheck() {
	s.check(s.a.editor.Text(), s.a.doc)
}

// underline draws a wavy line under the misspelt words; call it right
// after laying out the editor, with the same origin. The word being typed
// (the caret right after it) isn't marked yet.
func (s *spellChecker) underline(gtx layout.Context, size image.Point) {
	s.boxes = s.boxes[:0]
	caret, anchor := s.a.editor.Selection()
	from, to := s.nearCaret(caret)
	for i := from; i < to; i++ {
		if m := s.misses[i]; caret != anchor || caret != m.End {
			s.underlineMiss(gtx, i)
		}
	}
	s.listen(gtx, size)
}

// nearCaret returns the range of miss indices around the caret to
// underline; the visible ones are always among them.
func (s *spellChecker) nearCaret(caret int) (int, int) {
	i, _ := slices.BinarySearchFunc(s.misses, caret, missBefore)
	from := max(i-maxHighlights/2, 0)
	return from, min(from+maxHighlights, len(s.misses))
}

func missBefore(m spell.Miss, offset int) int { return m.Start - offset }

func (s *spellChecker) underlineMiss(gtx layout.Context, i int) {
	s.regions = s.a.editor.Regions(s.misses[i].Start, s.misses[i].End, s.regions[:0])
	for _, r := range s.regions {
		squiggle(gtx, r)
		s.boxes = append(s.boxes, missBox{r: r.Bounds, i: i})
	}
}

// squiggle draws a red wavy line just below the baseline of a region.
func squiggle(gtx layout.Context, r widget.Region) {
	dp := gtx.Metric.PxPerDp
	amp, step := 1*dp, 2*dp
	y := min(float32(r.Bounds.Max.Y-r.Baseline)+2.5*dp, float32(r.Bounds.Max.Y)-amp)
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(float32(r.Bounds.Min.X), y+amp))
	for i, x := 1, float32(r.Bounds.Min.X)+step; x <= float32(r.Bounds.Max.X); i, x = i+1, x+step {
		dy := amp
		if i%2 == 1 {
			dy = -amp
		}
		p.LineTo(f32.Pt(x, y+dy))
	}
	paint.FillShape(gtx.Ops, col.spellLine, clip.Stroke{Path: p.End(), Width: dp}.Op())
}

// listen asks for right-clicks on the editor, letting them through to it.
func (s *spellChecker) listen(gtx layout.Context, size image.Point) {
	area := clip.Rect{Max: size}.Push(gtx.Ops)
	pass := pointer.PassOp{}.Push(gtx.Ops)
	event.Op(gtx.Ops, &s.tag)
	pass.Pop()
	area.Pop()
}

// handleClick notes which misspelt word, if any, was right-clicked; it
// runs before the context menu opens in the same frame.
func (s *spellChecker) handleClick(gtx layout.Context) {
	s.clicked = nil
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &s.tag, Kinds: pointer.Press})
		if !ok {
			return
		}
		if e, isPtr := ev.(pointer.Event); isPtr && e.Buttons.Contain(pointer.ButtonSecondary) {
			s.clicked = s.missAt(e.Position.Round())
		}
	}
}

func (s *spellChecker) missAt(p image.Point) *spell.Miss {
	for _, b := range s.boxes {
		if p.In(b.r) && b.i < len(s.misses) {
			m := s.misses[b.i]
			return &m
		}
	}
	return nil
}

// menuItems are the context menu entries for a misspelt word: the
// suggestions, then adding it to the dictionary or ignoring it.
func (s *spellChecker) menuItems(m spell.Miss) []*command {
	var items []*command
	for _, w := range s.dict.Suggest(m.Word, maxSuggestions) {
		items = append(items, s.a.editCommand(w, "", 0, correction{m, w}.apply))
	}
	if len(items) == 0 {
		items = append(items, newCommand("No Suggestions", "", 0, doNothing))
	}
	return append(items, nil,
		newCommand("Add “"+m.Word+"” to Dictionary", "", 0, learnWord{s, m.Word, true}.run),
		newCommand("Ignore “"+m.Word+"”", "", 0, learnWord{s, m.Word, false}.run),
		nil)
}

func doNothing(layout.Context) {}

// correction replaces a misspelt word with a suggestion.
type correction struct {
	m    spell.Miss
	word string
}

func (c correction) apply(s edit.State) edit.State {
	return edit.ReplaceWord(s, c.m.Start, c.m.End, c.m.Word, c.word)
}

// learnWord accepts a word from now on: for good (saved in the settings)
// or until mdedit quits.
type learnWord struct {
	s    *spellChecker
	word string
	save bool
}

func (l learnWord) run(layout.Context) {
	l.s.dict.Add(l.word)
	l.s.recheck()
	if !l.save {
		return
	}
	st := loadSettings()
	st.Words = append(st.Words, l.word)
	if err := st.save(); err != nil {
		l.s.a.setStatus("Could not save the dictionary: %v", err)
	}
}

// cmdToggleSpelling shows or hides the spelling errors and remembers the
// choice.
func (a *App) cmdToggleSpelling(layout.Context) {
	a.spell.setOn(!a.spell.on)
	a.spell.recheck()
	st := loadSettings()
	st.NoSpelling = !a.spell.on
	if err := st.save(); err != nil {
		a.setStatus("Could not save the spelling setting: %v", err)
	}
}
