package ui

import (
	"strings"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"mdedit/internal/youtube"
)

// defaultVideoTitle is used when no title is given.
const defaultVideoTitle = "YouTube video"

// VideoDialog asks for a YouTube video ID (or link) and a title.
type VideoDialog struct {
	// Done receives the video ID and title, or an empty ID if cancelled.
	Done func(id, title string)

	th             *material.Theme
	id, title      widget.Editor
	insert, cancel widget.Clickable
	err            string
	focused        bool
}

func NewVideoDialog(done func(id, title string)) *VideoDialog {
	d := &VideoDialog{Done: done}
	for _, e := range []*widget.Editor{&d.id, &d.title} {
		e.SingleLine = true
		e.Submit = true
	}
	return d
}

func (d *VideoDialog) submit() {
	id, ok := youtube.ParseID(d.id.Text())
	if !ok {
		d.err = "That isn't a YouTube video ID or link."
		return
	}
	title := strings.TrimSpace(d.title.Text())
	if title == "" {
		title = defaultVideoTitle
	}
	d.Done(id, title)
}

func (d *VideoDialog) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	d.th = th
	if d.handleInput(gtx) {
		return layout.Dimensions{}
	}
	return modal(gtx, d.card)
}

// handleInput processes keys and clicks; it reports whether the dialog was
// cancelled.
func (d *VideoDialog) handleInput(gtx layout.Context) bool {
	if !d.focused {
		gtx.Execute(key.FocusCmd{Tag: &d.id})
		d.focused = true
	}
	name, pressed := keyPressed(gtx, key.NameTab, key.NameEscape)
	if (pressed && name == key.NameEscape) || d.cancel.Clicked(gtx) {
		d.Done("", "")
		return true
	}
	if pressed {
		d.switchField(gtx)
	}
	if d.submitted(gtx, &d.id) || d.submitted(gtx, &d.title) || d.insert.Clicked(gtx) {
		d.submit()
	}
	return false
}

func (d *VideoDialog) submitted(gtx layout.Context, e *widget.Editor) bool {
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

// switchField moves the focus between the two fields (Tab).
func (d *VideoDialog) switchField(gtx layout.Context) {
	next := &d.title
	if gtx.Focused(&d.title) {
		next = &d.id
	}
	gtx.Execute(key.FocusCmd{Tag: next})
}

func (d *VideoDialog) card(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(520))
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.UniformInset(20).Layout(gtx, d.content)
}

func (d *VideoDialog) content(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.H6(d.th, "Insert YouTube Video").Layout),
		vspace(14),
		layout.Rigid(d.caption("Video ID or link")),
		vspace(4),
		layout.Rigid(field{th: d.th, ed: &d.id, hint: "e.g. dQw4w9WgXcQ or https://youtu.be/…"}.layout),
		vspace(12),
		layout.Rigid(d.caption("Title (optional)")),
		vspace(4),
		layout.Rigid(field{th: d.th, ed: &d.title, hint: defaultVideoTitle}.layout),
		layout.Rigid(d.errorLine),
		vspace(16),
		layout.Rigid(d.buttons),
	)
}

func (d *VideoDialog) caption(text string) layout.Widget {
	return material.Body2(d.th, text).Layout
}

func (d *VideoDialog) errorLine(gtx layout.Context) layout.Dimensions {
	if d.err == "" {
		return layout.Dimensions{}
	}
	l := material.Body2(d.th, d.err)
	l.Color = col.errorColor
	return layout.Inset{Top: 10}.Layout(gtx, l.Layout)
}

func (d *VideoDialog) buttons(gtx layout.Context) layout.Dimensions {
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(button{th: d.th, click: &d.cancel, label: "Cancel"}.layout),
		hspace(8),
		layout.Rigid(button{th: d.th, click: &d.insert, label: "Insert", primary: true}.layout),
	)
}
