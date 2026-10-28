//go:build !js

package ui

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/io/key"
	"gioui.org/layout"

	"mdedit/internal/imgcache"
	"mdedit/internal/mdmodel"
)

// TestScreenshot renders the window offscreen to PNG files. Set MDEDIT_SHOT
// to a directory to run it; it is for checking the UI without a display.
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
	s := &shooter{h: h, w: w, dir: dir}
	s.previewScenes()
	s.findScenes()
	s.overlayScenes()
	s.contentScenes()
	s.darkScenes()
}

// darkScenes repeats the main scenes in the dark theme.
func (s *shooter) darkScenes() {
	a := s.h.a
	a.setDark(true)
	defer a.setDark(false)
	src, err := os.ReadFile("../mdmodel/testdata/TEST.md")
	if err != nil {
		s.h.t.Fatal(err)
	}
	s.scene("dark-split.png", string(src))
	a.preview.ScrollToBlock(a.doc.HeadingIndex("code-blocks"))
	a.bar.open = 2 // Format
	s.shot("dark-menu.png")
	a.bar.open = -1
	s.h.frame() // closing a menu takes a frame before shortcuts listen again
	s.h.press("R", key.ModShortcut)
	a.find.query.SetText("code")
	s.h.frame()
	s.h.press(key.NameReturn, 0)
	s.shot("dark-find.png")
	s.h.press(key.NameEscape, 0)
	s.scene("dark-highlight.png", highlightSample)
	s.scene("dark-spelling.png", spellingSample)
	s.darkMaths()
	a.tableDialog()
	s.shot("dark-dialog.png")
	a.dialog = nil
}

func (s *shooter) darkMaths() {
	if src, err := os.ReadFile("../mdmodel/testdata/maths.md"); err == nil {
		s.h.a.view = viewPreview
		s.scene("dark-maths.png", string(src))
		s.h.a.view = viewSplit
	}
	if src, err := os.ReadFile("../mdmodel/testdata/videos.md"); err == nil {
		s.scene("dark-videos.png", string(src))
	}
}

// shooter saves screenshots of the harness's window.
type shooter struct {
	h   *harness
	w   *headless.Window
	dir string
}

func (s *shooter) shot(name string) {
	for i := 0; i < 3; i++ {
		s.h.frame()
	}
	if err := s.w.Frame(&s.h.ops); err != nil {
		s.h.t.Fatal(err)
	}
	img := image.NewRGBA(image.Rectangle{Max: s.h.size})
	if err := s.w.Screenshot(img); err != nil {
		s.h.t.Fatal(err)
	}
	s.save(name, img)
}

func (s *shooter) save(name string, img image.Image) {
	f, err := os.Create(filepath.Join(s.dir, name))
	if err != nil {
		s.h.t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		s.h.t.Fatal(err)
	}
}

func (s *shooter) previewScenes() {
	s.shot("split.png")
	for _, id := range []string{"blockquotes", "lists", "code-blocks"} {
		s.h.a.preview.ScrollToBlock(s.h.a.doc.HeadingIndex(id))
		s.shot("preview-" + id + ".png")
	}
}

func (s *shooter) overlayScenes() {
	a := s.h.a
	a.bar.open = 2 // Format
	s.shot("menu-format.png")
	a.bar.open = -1
	a.tableDialog()
	s.shot("dialog-table.png")
	a.dialog = NewFileDialog("Open Markdown file", "../mdmodel/testdata", "", false, mdExts, ignorePath)
	s.shot("dialog.png")
	a.dialog = nil
}

func ignorePath(string) {}

// findScenes shows the replace bar with matches highlighted, and the Edit
// menu with the find commands.
func (s *shooter) findScenes() {
	a := s.h.a
	a.preview.ScrollToBlock(0)
	a.editor.SetCaret(0, 0)
	s.h.press("R", key.ModShortcut)
	a.find.query.SetText("markdown")
	a.find.repl.SetText("Markdown")
	s.h.frame()
	s.h.press(key.NameReturn, 0)
	s.h.press(key.NameReturn, 0)
	s.shot("find-replace.png")
	s.h.press(key.NameEscape, 0)
	a.bar.open = 1 // Edit
	s.shot("menu-edit.png")
	a.bar.open = -1
}

func (s *shooter) contentScenes() {
	s.scene("align.png", alignSample)
	s.scene("highlight.png", highlightSample)
	if src, err := os.ReadFile("../mdmodel/testdata/maths.md"); err == nil {
		s.h.a.view = viewPreview
		s.scene("maths.png", string(src))
		s.h.a.view = viewSplit
	}
	s.videoScenes()
	s.h.setText(tocSample, 0, 0)
	s.h.run(s.h.a.cmds.insertTOC)
	s.h.a.preview.ScrollToBlock(0)
	s.h.rightClick(700, 700)
	s.shot("toc.png")
	s.h.a.ctx.open = false
	s.spellingScene()
}

// spellingScene shows misspelt words underlined and the corrections for
// one of them.
func (s *shooter) spellingScene() {
	s.scene("spelling.png", spellingSample)
	s.h.rightClickMiss(1)
	s.shot("spelling-menu.png")
	s.h.a.ctx.open = false
}

// videoScenes shows YouTube stills in the preview (downloading them if
// needed) and the Insert Video dialog.
func (s *shooter) videoScenes() {
	src, err := os.ReadFile("../mdmodel/testdata/videos.md")
	if err != nil {
		s.h.t.Fatal(err)
	}
	s.h.setText(string(src), 0, 0)
	s.h.a.preview.ScrollToBlock(0)
	s.waitForImages(string(src))
	s.shot("videos.png")
	s.h.a.cmdVideo(layout.Context{})
	s.shot("dialog-video.png")
	s.h.a.dialog = nil
}

// waitForImages gives background downloads of the document's YouTube
// stills up to 10 seconds.
func (s *shooter) waitForImages(doc string) {
	deadline := time.Now().Add(10 * time.Second)
	for !s.imagesCached(doc) && time.Now().Before(deadline) {
		s.h.frame()
		time.Sleep(100 * time.Millisecond)
	}
}

func (s *shooter) imagesCached(doc string) bool {
	for _, b := range mdmodel.Parse([]byte(doc)).Blocks {
		if _, ok := imgcache.Path(b.Src); b.Video && !ok {
			return false
		}
	}
	return true
}

// scene shows a document from the top and saves a screenshot.
func (s *shooter) scene(name, doc string) {
	s.h.setText(doc, 0, 0)
	s.h.a.preview.ScrollToBlock(0)
	s.shot(name)
}

const alignSample = "# Aligned\n\n<div align=\"center\">\n\nCentred **bold** *italic* <u>underlined</u> text.\n\n</div>\n\n" +
	"<div align=\"right\">\n\nRight aligned.\n\n</div>\n\nLeft again."

const highlightSample = "```bash\n# build and run\nfor f in *.md; do\n  mdedit -pdf \"$f\" -o \"${f%.md}.pdf\"\ndone\n```\n\n" +
	"```C#\nusing System;\n\npublic class Hello {\n    // Entry point\n    public static void Main(string[] args) {\n" +
	"        Console.WriteLine($\"Hello {args.Length}\");\n    }\n}\n```\n\n" +
	"```go\nfunc fib(n int) int {\n\tif n < 2 {\n\t\treturn n // base case\n\t}\n\treturn fib(n-1) + fib(n-2)\n}\n```\n"

const tocSample = "# Guide\n\n## Install\n\n### Linux\n\n### FreeBSD\n\n## Usage\n\n### Shortcuts\n\n## Licence\n"

const spellingSample = "# Speling\n\nThis sentense has a fwe mistakes, but `codde`, $x^2$ maths and\n" +
	"[links](https://exampel.com) are left alone. Recieve the definately seperate words.\n"
