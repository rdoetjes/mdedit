package ui

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/io/input"
	"gioui.org/io/key"
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

func TestEnterContinuesLists(t *testing.T) {
	h := newHarness(t, "")
	if !h.router.Source().Focused(&h.a.editor) {
		t.Fatal("editor not focused")
	}
	h.setText("1. one", 6, 6)
	h.press(key.NameReturn, 0)
	h.expect("1. one\n2. ")
	h.press(key.NameReturn, 0)
	h.expect("1. one\n\n")

	h.setText("- a", 3, 3)
	h.press(key.NameReturn, 0)
	h.expect("- a\n- ")

	h.setText("plain", 5, 5)
	h.press(key.NameReturn, 0)
	h.expect("plain\n")
}

func TestShortcuts(t *testing.T) {
	h := newHarness(t, "")
	h.setText("word", 0, 4)
	h.press("B", key.ModShortcut)
	h.expect("**word**")
	h.press("B", key.ModShortcut)
	h.expect("word")
	h.press("1", key.ModShortcut)
	h.expect("# word")
}

func TestTableDialog(t *testing.T) {
	h := newHarness(t, "")
	h.setText("Intro", 5, 5)
	h.a.tableDialog()
	h.frame()
	d := h.a.dialog.(*TableDialog)
	d.cols.SetText("2")
	d.rows.SetText("1")
	d.submit()
	h.frame()
	h.frame()
	h.expect("Intro\n\n|     |     |\n| --- | --- |\n|     |     |\n")
	if h.a.dialog != nil {
		t.Error("dialog still open")
	}
}

// TestScreenshot renders the window offscreen. Set MDEDIT_SHOT to a directory
// to run it; useful for checking the UI without a display.
func TestScreenshot(t *testing.T) {
	dir := os.Getenv("MDEDIT_SHOT")
	if dir == "" {
		t.Skip("MDEDIT_SHOT not set")
	}
	h := newHarness(t, "../mdmodel/testdata/TEST.md")
	w, err := headless.NewWindow(h.size.X, h.size.Y)
	if err != nil {
		t.Fatal(err)
	}
	shot := func(name string) {
		for i := 0; i < 3; i++ {
			h.frame()
		}
		if err := w.Frame(&h.ops); err != nil {
			t.Fatal(err)
		}
		img := image.NewRGBA(image.Rectangle{Max: h.size})
		if err := w.Screenshot(img); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		png.Encode(f, img)
	}
	shot("split.png")
	for _, id := range []string{"blockquotes", "lists", "code-blocks"} {
		h.a.preview.ScrollToBlock(h.a.doc.HeadingIndex(id))
		shot("preview-" + id + ".png")
	}
	h.a.bar.open = 2 // Format
	shot("menu-format.png")
	h.a.bar.open = -1
	h.a.tableDialog()
	shot("dialog-table.png")
	h.a.dialog = nil

	h.setText("# Aligned\n\n<div align=\"center\">\n\nCentred **bold** *italic* <u>underlined</u> text.\n\n</div>\n\n<div align=\"right\">\n\nRight aligned.\n\n</div>\n\nLeft again.", 0, 0)
	h.a.preview.ScrollToBlock(0)
	shot("align.png")
	h.a.dialog = NewFileDialog("Open Markdown file", "../mdmodel/testdata", "", false, mdExts, func(string) {})
	shot("dialog.png")
	h.a.dialog = nil

	h.setText("```bash\n# build and run\nfor f in *.md; do\n  mdedit -pdf \"$f\" -o \"${f%.md}.pdf\"\ndone\n```\n\n"+
		"```C#\nusing System;\n\npublic class Hello {\n    // Entry point\n    public static void Main(string[] args) {\n        Console.WriteLine($\"Hello {args.Length}\");\n    }\n}\n```\n\n"+
		"```go\nfunc fib(n int) int {\n\tif n < 2 {\n\t\treturn n // base case\n\t}\n\treturn fib(n-1) + fib(n-2)\n}\n```\n", 0, 0)
	h.a.preview.ScrollToBlock(0)
	shot("highlight.png")
}


func TestMacNavigation(t *testing.T) {
	if key.ModShortcut != key.ModCommand {
		t.Skip("macOS key bindings")
	}
	h := newHarness(t, "")
	caret := func(want int) {
		t.Helper()
		if c, _ := h.a.editor.Selection(); c != want {
			t.Errorf("caret at %d, want %d", c, want)
		}
	}
	cmd := key.ModCommand
	h.setText("abc\ndefgh", 6, 6)
	h.press(key.NameLeftArrow, cmd)
	caret(4)
	h.press(key.NameRightArrow, cmd)
	caret(9)
	h.press(key.NameUpArrow, cmd)
	caret(0)
	h.press(key.NameDownArrow, cmd|key.ModShift)
	if s, e := h.a.editor.Selection(); s != 9 || e != 0 {
		t.Errorf("selection %d-%d, want 9-0", s, e)
	}
	h.setText("abc\ndefgh", 6, 6)
	h.press(key.NameDeleteBackward, cmd)
	h.expect("abc\nfgh")

	// A long line wraps; ⌘← and ⌘→ stay on the screen line.
	long := strings.Repeat("word ", 400)
	h.setText(long, 1000, 1000)
	h.press(key.NameLeftArrow, cmd)
	start, _ := h.a.editor.Selection()
	if start <= 0 || start >= 1000 {
		t.Errorf("⌘← in wrapped line went to %d, want a screen line start in (0, 1000)", start)
	}
	h.press(key.NameRightArrow, cmd)
	end, _ := h.a.editor.Selection()
	if end <= 1000 || end >= len(long) {
		t.Errorf("⌘→ in wrapped line went to %d, want a screen line end in (1000, %d)", end, len(long))
	}
	t.Logf("screen line: %d-%d", start, end)
}

func TestTabKey(t *testing.T) {
	h := newHarness(t, "")
	h.setText("- a\n- b", 7, 7)
	h.press(key.NameTab, 0)
	h.expect("- a\n    - b")
	if !h.router.Source().Focused(&h.a.editor) {
		t.Error("Tab moved focus out of the editor")
	}
	h.press(key.NameTab, key.ModShift)
	h.expect("- a\n- b")
	h.setText("x", 1, 1)
	h.a.editor.Insert("\tpasted")
	h.frame()
	h.frame()
	h.expect("x   pasted")
}
