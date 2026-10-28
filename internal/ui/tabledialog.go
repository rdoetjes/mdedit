package ui

import (
	"strconv"
	"strings"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// TableDialog asks for the size of a new table.
type TableDialog struct {
	// Done receives the column and body row counts, or (0, 0) if cancelled.
	Done func(cols, rows int)

	th             *material.Theme
	cols, rows     widget.Editor
	insert, cancel widget.Clickable
	err            string
	focused        bool
}

const maxTableCols, maxTableRows = 30, 200

func NewTableDialog(done func(cols, rows int)) *TableDialog {
	d := &TableDialog{Done: done}
	for _, e := range []*widget.Editor{&d.cols, &d.rows} {
		e.SingleLine = true
		e.Submit = true
		e.Filter = "0123456789"
		e.MaxLen = 3
		e.SetText("3")
	}
	d.cols.SetCaret(0, 1) // select the default so typing replaces it
	return d
}

func (d *TableDialog) submit() {
	cols, err1 := strconv.Atoi(strings.TrimSpace(d.cols.Text()))
	rows, err2 := strconv.Atoi(strings.TrimSpace(d.rows.Text()))
	switch {
	case err1 != nil || cols < 1 || cols > maxTableCols:
		d.err = "Columns must be between 1 and " + strconv.Itoa(maxTableCols) + "."
	case err2 != nil || rows < 0 || rows > maxTableRows:
		d.err = "Rows must be between 0 and " + strconv.Itoa(maxTableRows) + "."
	default:
		d.Done(cols, rows)
	}
}

func (d *TableDialog) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	d.th = th
	if d.handleInput(gtx) {
		return layout.Dimensions{}
	}
	return modal(gtx, d.card)
}

// handleInput processes keys and clicks; it reports whether the dialog was
// cancelled.
func (d *TableDialog) handleInput(gtx layout.Context) bool {
	if !d.focused {
		gtx.Execute(key.FocusCmd{Tag: &d.cols})
		d.focused = true
	}
	d.handleSubmit(gtx, &d.cols)
	d.handleSubmit(gtx, &d.rows)
	name, pressed := keyPressed(gtx, key.NameTab, key.NameEscape)
	if (pressed && name == key.NameEscape) || d.cancel.Clicked(gtx) {
		d.Done(0, 0)
		return true
	}
	if pressed {
		d.switchField(gtx)
	}
	if d.insert.Clicked(gtx) {
		d.submit()
	}
	return false
}

func (d *TableDialog) handleSubmit(gtx layout.Context, e *widget.Editor) {
	for {
		ev, ok := e.Update(gtx)
		if !ok {
			return
		}
		if _, submit := ev.(widget.SubmitEvent); submit {
			d.submit()
		}
	}
}

// switchField moves the focus between the two fields (Tab).
func (d *TableDialog) switchField(gtx layout.Context) {
	next := &d.rows
	if gtx.Focused(&d.rows) {
		next = &d.cols
	}
	next.SetCaret(next.Len(), 0)
	gtx.Execute(key.FocusCmd{Tag: next})
}

func (d *TableDialog) card(gtx layout.Context) layout.Dimensions {
	return layout.UniformInset(20).Layout(gtx, d.content)
}

func (d *TableDialog) content(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.H6(d.th, "Insert Table").Layout),
		vspace(14),
		layout.Rigid(numberField{th: d.th, label: "Columns:", ed: &d.cols}.layout),
		layout.Rigid(numberField{th: d.th, label: "Rows:", ed: &d.rows}.layout),
		layout.Rigid(d.errorLine),
		vspace(6),
		layout.Rigid(d.buttons),
	)
}

func (d *TableDialog) errorLine(gtx layout.Context) layout.Dimensions {
	if d.err == "" {
		return layout.Dimensions{}
	}
	l := material.Body2(d.th, d.err)
	l.Color = col.errorColor
	return layout.Inset{Bottom: 10}.Layout(gtx, l.Layout)
}

func (d *TableDialog) buttons(gtx layout.Context) layout.Dimensions {
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(button{th: d.th, click: &d.cancel, label: "Cancel"}.layout),
		hspace(8),
		layout.Rigid(button{th: d.th, click: &d.insert, label: "Insert", primary: true}.layout),
	)
}

// numberField is a labelled, narrow numeric input.
type numberField struct {
	th    *material.Theme
	label string
	ed    *widget.Editor
}

func (f numberField) layout(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Bottom: 10}.Layout(gtx, f.row)
}

func (f numberField) row(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(f.caption),
		layout.Rigid(f.input),
	)
}

func (f numberField) caption(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Dp(90)
	return material.Body1(f.th, f.label).Layout(gtx)
}

func (f numberField) input(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Dp(80)
	gtx.Constraints.Max.X = gtx.Dp(80)
	return framed(gtx, f.editor)
}

func (f numberField) editor(gtx layout.Context) layout.Dimensions {
	return layout.UniformInset(6).Layout(gtx, material.Editor(f.th, f.ed, "").Layout)
}
