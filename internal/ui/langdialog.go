package ui

import (
	"slices"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"mdedit/internal/spell"
)

// LanguageDialog lets the user tick the spelling languages.
type LanguageDialog struct {
	// done receives the chosen codes in catalogue order, or nil if
	// cancelled.
	done func(codes []string)

	th     *material.Theme
	checks []widget.Bool
	have   []bool // the dictionary is on this machine
	list   widget.List
	ok     widget.Clickable
	cancel widget.Clickable
	err    string
}

func newLanguageDialog(chosen []string, have func(code string) bool, done func([]string)) *LanguageDialog {
	d := &LanguageDialog{done: done}
	d.list.Axis = layout.Vertical
	d.checks = make([]widget.Bool, len(spell.Languages))
	d.have = make([]bool, len(spell.Languages))
	for i, l := range spell.Languages {
		d.checks[i].Value = slices.Contains(chosen, l.Code)
		d.have[i] = have(l.Code)
	}
	return d
}

func (d *LanguageDialog) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	d.th = th
	if d.handleInput(gtx) {
		return layout.Dimensions{}
	}
	return modal(gtx, d.card)
}

// handleInput reports whether the dialog closed.
func (d *LanguageDialog) handleInput(gtx layout.Context) bool {
	if _, esc := keyPressed(gtx, key.NameEscape); esc || d.cancel.Clicked(gtx) {
		d.done(nil)
		return true
	}
	if !d.ok.Clicked(gtx) {
		return false
	}
	codes := d.chosen()
	if len(codes) == 0 {
		d.err = "Choose at least one language, or use Edit → Hide Spelling Errors."
		return false
	}
	d.done(codes)
	return true
}

func (d *LanguageDialog) chosen() []string {
	var codes []string
	for i, l := range spell.Languages {
		if d.checks[i].Value {
			codes = append(codes, l.Code)
		}
	}
	return codes
}

func (d *LanguageDialog) card(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(480))
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.UniformInset(20).Layout(gtx, d.content)
}

func (d *LanguageDialog) content(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.H6(d.th, "Spelling Languages").Layout),
		vspace(8),
		layout.Rigid(d.caption("A word is correct if any ticked language knows it. Dictionaries "+
			"that aren't on this computer yet are downloaded when you click OK.")),
		vspace(10),
		layout.Rigid(d.languageList),
		vspace(8),
		layout.Rigid(d.caption("Dictionaries from LibreOffice and Hunspell, via "+
			"github.com/wooorm/dictionaries; each has its own licence.")),
		layout.Rigid(d.errorLine),
		vspace(14),
		layout.Rigid(d.buttons),
	)
}

func (d *LanguageDialog) caption(text string) layout.Widget {
	l := material.Caption(d.th, text)
	l.Color = col.statusFg
	return l.Layout
}

// languageList is a framed, scrolling list of the languages.
func (d *LanguageDialog) languageList(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Max.Y = min(gtx.Constraints.Max.Y, gtx.Dp(340))
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	return framed(gtx, d.listInside)
}

func (d *LanguageDialog) listInside(gtx layout.Context) layout.Dimensions {
	return layout.UniformInset(4).Layout(gtx, d.listView)
}

func (d *LanguageDialog) listView(gtx layout.Context) layout.Dimensions {
	return material.List(d.th, &d.list).Layout(gtx, len(spell.Languages), d.row)
}

// row is one language: its check box and whether it must be downloaded.
func (d *LanguageDialog) row(gtx layout.Context, i int) layout.Dimensions {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, material.CheckBox(d.th, &d.checks[i], spell.Languages[i].Name).Layout),
		layout.Rigid(d.caption(d.status(i))),
		hspace(10),
	)
}

func (d *LanguageDialog) status(i int) string {
	switch {
	case spell.Languages[i].Code == spell.BuiltIn:
		return "built in"
	case d.have[i]:
		return "downloaded"
	}
	return ""
}

func (d *LanguageDialog) errorLine(gtx layout.Context) layout.Dimensions {
	if d.err == "" {
		return layout.Dimensions{}
	}
	l := material.Body2(d.th, d.err)
	l.Color = col.errorColor
	return layout.Inset{Top: 8}.Layout(gtx, l.Layout)
}

func (d *LanguageDialog) buttons(gtx layout.Context) layout.Dimensions {
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(button{th: d.th, click: &d.cancel, label: "Cancel"}.layout),
		hspace(8),
		layout.Rigid(button{th: d.th, click: &d.ok, label: "OK", primary: true}.layout),
	)
}
