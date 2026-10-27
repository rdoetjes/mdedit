package ui

import (
	"image/color"
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
	if !d.focused {
		gtx.Execute(key.FocusCmd{Tag: &d.cols})
		d.focused = true
	}
	for _, e := range []*widget.Editor{&d.cols, &d.rows} {
		for {
			ev, ok := e.Update(gtx)
			if !ok {
				break
			}
			if _, ok := ev.(widget.SubmitEvent); ok {
				d.submit()
			}
		}
	}
	// Tab moves between the two fields.
	for {
		ev, ok := gtx.Event(key.Filter{Name: key.NameTab, Optional: key.ModShift}, key.Filter{Name: key.NameEscape})
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		if e.Name == key.NameEscape {
			d.Done(0, 0)
			return layout.Dimensions{}
		}
		next := &d.rows
		if gtx.Focused(&d.rows) {
			next = &d.cols
		}
		next.SetCaret(next.Len(), 0)
		gtx.Execute(key.FocusCmd{Tag: next})
	}
	if d.cancel.Clicked(gtx) {
		d.Done(0, 0)
		return layout.Dimensions{}
	}
	if d.insert.Clicked(gtx) {
		d.submit()
	}

	field := func(label string, e *widget.Editor) layout.FlexChild {
		return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(90)
						return material.Body1(th, label).Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(80)
						gtx.Constraints.Max.X = gtx.Dp(80)
						return widget.Border{Color: borderColor, Width: 1, CornerRadius: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(6).Layout(gtx, material.Editor(th, e, "").Layout)
						})
					}),
				)
			})
		})
	}
	return modal(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(20).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(material.H6(th, "Insert Table").Layout),
				layout.Rigid(layout.Spacer{Height: 14}.Layout),
				field("Columns:", &d.cols),
				field("Rows:", &d.rows),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if d.err == "" {
						return layout.Dimensions{}
					}
					l := material.Body2(th, d.err)
					l.Color = color.NRGBA{R: 0xcf, G: 0x22, B: 0x2e, A: 0xff}
					return layout.Inset{Bottom: 10}.Layout(gtx, l.Layout)
				}),
				layout.Rigid(layout.Spacer{Height: 6}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return smallButton(gtx, th, &d.cancel, "Cancel")
						}),
						layout.Rigid(layout.Spacer{Width: 8}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return primaryButton(gtx, th, &d.insert, "Insert")
						}),
					)
				}),
			)
		})
	})
}
