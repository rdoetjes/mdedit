//go:build !js

package ui

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// harness drives an App without a window, feeding events through Gio's
// input router like the real event loop does.
type harness struct {
	t      *testing.T
	a      *App
	router input.Router
	ops    op.Ops
	size   image.Point
}

func newHarness(t *testing.T, path string) *harness {
	a, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, a: a, size: image.Pt(2560, 1640)}
	h.frame()
	h.frame() // let the editor take focus
	return h
}

func (h *harness) frame() {
	h.ops.Reset()
	gtx := layout.Context{
		Ops:         &h.ops,
		Now:         time.Now(),
		Metric:      unit.Metric{PxPerDp: 2, PxPerSp: 2},
		Constraints: layout.Exact(h.size),
		Source:      h.router.Source(),
	}
	h.a.layout(gtx)
	h.router.Frame(&h.ops)
}

func (h *harness) press(name key.Name, mods key.Modifiers) {
	h.router.Queue(
		key.Event{Name: name, Modifiers: mods, State: key.Press},
		key.Event{Name: name, Modifiers: mods, State: key.Release},
	)
	h.frame()
	h.frame()
}

func (h *harness) rightClick(x, y float32) {
	at := f32.Pt(x, y)
	h.router.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonSecondary, Position: at},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: at},
	)
	h.frame()
	h.frame()
}

// scroll turns the mouse wheel by dy pixels at (x, y).
func (h *harness) scroll(x, y, dy float32) {
	h.router.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(x, y), Scroll: f32.Pt(0, dy)})
	h.frame()
	h.frame()
}

// run runs a command as if chosen from a menu.
func (h *harness) run(c *command) {
	h.a.ctx.open = false
	h.a.pendingCmd = c
	h.frame()
	h.frame()
}

func (h *harness) setText(s string, start, end int) {
	h.a.editor.SetText(s)
	h.a.editor.SetCaret(start, end)
	h.a.reparse()
	h.frame()
}

func (h *harness) expect(want string) {
	h.t.Helper()
	if got := h.a.editor.Text(); got != want {
		h.t.Errorf("editor text:\n got %q\nwant %q", got, want)
	}
}

func (h *harness) expectCaret(want int) {
	h.t.Helper()
	if c, _ := h.a.editor.Selection(); c != want {
		h.t.Errorf("caret at %d, want %d", c, want)
	}
}

func (h *harness) expectSelection(caret, anchor int) {
	h.t.Helper()
	if c, a := h.a.editor.Selection(); c != caret || a != anchor {
		h.t.Errorf("selection %d-%d, want %d-%d", c, a, caret, anchor)
	}
}
