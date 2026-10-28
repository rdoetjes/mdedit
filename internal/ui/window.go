package ui

import (
	"image"
	"strings"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/widget"

	"mdedit/internal/edit"
)

// contextMenu is the right-click menu over the editor and preview.
type contextMenu struct {
	tag   int // pointer event target for right-clicks
	open  bool
	at    image.Point
	popup popup
	// keepSel holds the editor selection across a right-click, which the
	// editor would otherwise treat as a click that moves the caret.
	keepSel *[2]int
}

// layout draws one frame.
func (a *App) layout(gtx layout.Context) {
	a.handleInput(gtx)
	a.handleEditorEvents(gtx)
	a.handleRightClick(gtx)
	a.updateTitle()
	size := gtx.Constraints.Max
	a.layoutWindow(gtx)
	a.restoreSelection()
	gtx.Constraints = layout.Exact(size)
	a.layoutOverlay(gtx)
}

// handleInput handles shortcuts, toolbar clicks and queued work, while no
// dialog is open.
func (a *App) handleInput(gtx layout.Context) {
	if a.dialog != nil {
		return
	}
	if a.bar.open < 0 {
		a.handleShortcuts(gtx)
	}
	a.handleEnter(gtx)
	a.handleTab(gtx)
	a.handleMacKeys(gtx)
	a.handleToolbar(gtx)
	a.find.handleInput(gtx)
	a.runPending(gtx)
}

func (a *App) handleToolbar(gtx layout.Context) {
	for _, c := range a.toolbarCmds {
		if c != nil && c.click.Clicked(gtx) {
			c.run(gtx)
		}
	}
}

func (a *App) runPending(gtx layout.Context) {
	if f := a.pendingEdit; f != nil {
		a.pendingEdit = nil
		a.applyEdit(gtx, f)
	}
	if c := a.pendingCmd; c != nil {
		a.pendingCmd = nil
		c.run(gtx)
	}
}

func (a *App) handleEditorEvents(gtx layout.Context) {
	for {
		ev, ok := a.editor.Update(gtx)
		if !ok {
			return
		}
		if _, changed := ev.(widget.ChangeEvent); changed {
			a.editorChanged(gtx)
		}
	}
}

func (a *App) editorChanged(gtx layout.Context) {
	if strings.ContainsRune(a.editor.Text(), '\t') {
		a.applyEdit(gtx, edit.ExpandTabs) // pasted text
	}
	a.reparse()
}

// handleRightClick opens the context menu on a right-click in the editor or
// preview.
func (a *App) handleRightClick(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &a.ctx.tag, Kinds: pointer.Press})
		if !ok {
			return
		}
		if e, isPtr := ev.(pointer.Event); isPtr && e.Buttons.Contain(pointer.ButtonSecondary) && a.dialog == nil {
			a.openContextMenu(e.Position.Round())
		}
	}
}

func (a *App) openContextMenu(at image.Point) {
	a.ctx.open = true
	a.ctx.at = at.Add(image.Pt(0, a.barH+a.toolH))
	a.bar.open = -1
	s, e := a.editor.Selection()
	a.ctx.keepSel = &[2]int{s, e}
}

func (a *App) restoreSelection() {
	if a.ctx.keepSel != nil {
		a.editor.SetCaret(a.ctx.keepSel[0], a.ctx.keepSel[1])
		a.ctx.keepSel = nil
	}
}

func (a *App) layoutWindow(gtx layout.Context) {
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(a.menuBarRow),
		layout.Rigid(a.toolbarRow),
		layout.Flexed(1, a.body),
		layout.Rigid(a.statusBar),
	)
}

func (a *App) menuBarRow(gtx layout.Context) layout.Dimensions {
	d := a.bar.Layout(gtx, a.th, a.menus, a.dialog == nil)
	a.barH = d.Size.Y
	return d
}

func (a *App) toolbarRow(gtx layout.Context) layout.Dimensions {
	d := a.toolbar(gtx)
	a.toolH = d.Size.Y
	return d
}

// layoutOverlay draws whatever floats over the window: a dialog, the
// context menu or an open menu.
func (a *App) layoutOverlay(gtx layout.Context) {
	switch {
	case a.dialog != nil:
		a.ctx.open = false
		a.dialog.Layout(gtx, a.th)
	case a.ctx.open:
		a.layoutContextMenu(gtx)
	default:
		a.bar.Overlay(gtx, a.th, a.menus)
	}
}

func (a *App) layoutContextMenu(gtx layout.Context) {
	chosen, closed := a.ctx.popup.Layout(gtx, a.th, a.contextItems(), a.ctx.at, image.Rectangle{})
	if closed {
		a.ctx.open = false
	}
	if chosen != nil {
		chosen.run(gtx)
	}
}

// body lays out the panes and watches for right-clicks over them, letting
// the events through to the editor and preview as well.
func (a *App) body(gtx layout.Context) layout.Dimensions {
	dims := a.panes(gtx)
	area := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
	pass := pointer.PassOp{}.Push(gtx.Ops)
	event.Op(gtx.Ops, &a.ctx.tag)
	pass.Pop()
	area.Pop()
	return dims
}
