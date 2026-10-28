package ui

import (
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// ConfirmDialog asks a question with up to three answers.
type ConfirmDialog struct {
	Message string
	Buttons []string         // the last button is the default action
	Done    func(choice int) // index into Buttons, or -1 if dismissed

	th     *material.Theme
	clicks []widget.Clickable
}

func (c *ConfirmDialog) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	c.th = th
	if len(c.clicks) != len(c.Buttons) {
		c.clicks = make([]widget.Clickable, len(c.Buttons))
	}
	if choice, ok := c.choice(gtx); ok {
		c.Done(choice)
		return layout.Dimensions{}
	}
	return modal(gtx, c.card)
}

// choice reports the answer given by keyboard (Escape dismisses, Return
// picks the default) or by clicking a button.
func (c *ConfirmDialog) choice(gtx layout.Context) (int, bool) {
	if name, ok := keyPressed(gtx, key.NameEscape, key.NameReturn); ok {
		if name == key.NameEscape {
			return -1, true
		}
		return len(c.Buttons) - 1, true
	}
	for i := range c.clicks {
		if c.clicks[i].Clicked(gtx) {
			return i, true
		}
	}
	return 0, false
}

func (c *ConfirmDialog) card(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(460))
	return layout.UniformInset(20).Layout(gtx, c.content)
}

func (c *ConfirmDialog) content(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(material.Body1(c.th, c.Message).Layout),
		vspace(16),
		layout.Rigid(c.buttons),
	)
}

func (c *ConfirmDialog) buttons(gtx layout.Context) layout.Dimensions {
	var children []layout.FlexChild
	for i, label := range c.Buttons {
		if i > 0 {
			children = append(children, hspace(8))
		}
		b := button{th: c.th, click: &c.clicks[i], label: label, primary: i == len(c.Buttons)-1}
		children = append(children, layout.Rigid(b.layout))
	}
	return layout.Flex{}.Layout(gtx, children...)
}
