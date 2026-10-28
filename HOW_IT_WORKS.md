# How mdedit works

## TOC

<!-- toc -->

- [How mdedit works](#how-mdedit-works)
  - [TOC](#toc)
  - [1. The big picture](#1-the-big-picture)
    - [Packages](#packages)
  - [2. The data that flows through the layers](#2-the-data-that-flows-through-the-layers)
    - [edit.State — text plus selection (edit/edit.go)](#editstate--text-plus-selection-editeditgo)
    - [mdmodel.Block and Span — the parsed document (mdmodel/model.go)](#mdmodelblock-and-span--the-parsed-document-mdmodelmodelgo)
    - [render.Layout and render.Prim — what to draw where (render/render.go)](#renderlayout-and-renderprim--what-to-draw-where-renderrendergo)
    - [render.Ctx — the device the layout is for (render/render.go)](#renderctx--the-device-the-layout-is-for-renderrendergo)
  - [3. The frame loop (Gio is immediate-mode)](#3-the-frame-loop-gio-is-immediate-mode)
  - [4. Walkthrough: clicking Format → Bold](#4-walkthrough-clicking-format--bold)
  - [5. Walkthrough: a keystroke to pixels in the preview](#5-walkthrough-a-keystroke-to-pixels-in-the-preview)
  - [6. From a character to pixels](#6-from-a-character-to-pixels)
    - [6.1 The font files](#61-the-font-files)
    - [6.2 Step 1 — character to glyph number (cmap)](#62-step-1--character-to-glyph-number-cmap)
    - [6.3 Step 2 — shaping: which glyphs, and where](#63-step-2--shaping-which-glyphs-and-where)
    - [6.4 Step 3 — glyph number to bytes (loca)](#64-step-3--glyph-number-to-bytes-loca)
    - [6.5 Step 4 — bytes to points (glyf)](#65-step-4--bytes-to-points-glyf)
    - [6.6 Step 5 — points to curves (implied midpoints)](#66-step-5--points-to-curves-implied-midpoints)
    - [6.7 Step 6 — curves to a path in pixels](#67-step-6--curves-to-a-path-in-pixels)
    - [6.8 Step 7 — the GPU: coverage, then colour](#68-step-7--the-gpu-coverage-then-colour)
    - [6.9 Maths glyphs: mdedit does steps 1–6 itself](#69-maths-glyphs-mdedit-does-steps-16-itself)
    - [6.10 Images: the same pipeline, a different paint](#610-images-the-same-pipeline-a-different-paint)
      - [What an image is](#what-an-image-is)
      - [Step A — uploaded once as a texture](#step-a--uploaded-once-as-a-texture)
      - [Step B — a rectangle, placed by a transform](#step-b--a-rectangle-placed-by-a-transform)
      - [Step C — every screen pixel looks up its colour](#step-c--every-screen-pixel-looks-up-its-colour)
      - [Paths or pixels: why both exist](#paths-or-pixels-why-both-exist)
    - [6.11 The PDF: the viewer rasterises](#611-the-pdf-the-viewer-rasterises)
  - [7. Images and YouTube videos](#7-images-and-youtube-videos)
    - [7.1 How they are written](#71-how-they-are-written)
    - [7.2 Inserting a video](#72-inserting-a-video)
    - [7.3 Parsing: an image block](#73-parsing-an-image-block)
    - [7.4 Finding the pixels: local files and the download cache](#74-finding-the-pixels-local-files-and-the-download-cache)
    - [7.5 Layout: size, alignment, placeholder, play button](#75-layout-size-alignment-placeholder-play-button)
    - [7.6 The preview: decode, cache, arrive late, click](#76-the-preview-decode-cache-arrive-late-click)
    - [7.7 The PDF](#77-the-pdf)
    - [7.8 When an image doesn't show](#78-when-an-image-doesnt-show)
  - [8. The browser version (WebAssembly)](#8-the-browser-version-webassembly)
    - [8.1 The platform seam in internal/ui](#81-the-platform-seam-in-internalui)
    - [8.2 Asynchronous results: picker, file reading, PDF export](#82-asynchronous-results-picker-file-reading-pdf-export)
    - [8.3 Storage](#83-storage)
    - [8.4 The web page and keys](#84-the-web-page-and-keys)
    - [8.5 Building, serving, installing](#85-building-serving-installing)
    - [8.6 Adding something that touches the outside world](#86-adding-something-that-touches-the-outside-world)
  - [9. The layers in detail](#9-the-layers-in-detail)
    - [9.1 internal/ui — the window](#91-internalui--the-window)
    - [9.2 internal/edit — editing commands as pure functions](#92-internaledit--editing-commands-as-pure-functions)
    - [9.3 internal/mdmodel — Markdown to blocks](#93-internalmdmodel--markdown-to-blocks)
    - [9.4 internal/render — blocks to primitives](#94-internalrender--blocks-to-primitives)
    - [9.5 internal/pdfout — primitives to PDF](#95-internalpdfout--primitives-to-pdf)
    - [9.6 internal/tex — maths](#96-internaltex--maths)
    - [9.7 internal/imgcache and internal/youtube](#97-internalimgcache-and-internalyoutube)
  - [10. Recipes: where to add things](#10-recipes-where-to-add-things)
    - [Add an editing command (menu item, shortcut, maybe a toolbar icon)](#add-an-editing-command-menu-item-shortcut-maybe-a-toolbar-icon)
    - [Add a dialog](#add-a-dialog)
    - [Add a Markdown block type](#add-a-markdown-block-type)
    - [Add an inline style or inline element](#add-an-inline-style-or-inline-element)
    - [Add a new primitive kind](#add-a-new-primitive-kind)
    - [Add a TeX command](#add-a-tex-command)
    - [Add a keyboard behaviour inside the editor](#add-a-keyboard-behaviour-inside-the-editor)
  - [11. Finding bugs](#11-finding-bugs)
    - [Symptom → where to look](#symptom--where-to-look)
    - [Tools](#tools)
    - [Gotchas](#gotchas)
  - [12. Code style](#12-code-style)

<!-- /toc -->

This document explains how mdedit is put together: which layer does what,
how a click or a keystroke travels through the code, where to add new
features, and where to look when something breaks.

Code is referred to as `package/file.go: Function` rather than by line
number, because line numbers drift. Use your editor's "go to symbol" to jump
there.

---

## 1. The big picture

mdedit turns Markdown text into pictures twice — once on screen and once as
PDF — from **one shared layout**. Everything is organised as a pipeline of
layers, each knowing only about the layer below it:

```
                ┌──────────────────────────────────────────────┐
  keyboard,     │ ui        Gio window: editor, preview, menus,│
  mouse  ──────►│           toolbar, dialogs, find bar         │
                └───────┬───────────────────────┬──────────────┘
                        │ pure text edits       │ text changed
                        ▼                       ▼
                ┌──────────────┐        ┌──────────────────────┐
                │ edit         │        │ mdmodel              │
                │ bold, lists, │        │ Markdown → []Block   │
                │ TOC, search… │        │ (goldmark + maths +  │
                └──────────────┘        │  syntax highlighting)│
                                        └──────────┬───────────┘
                                                   ▼
                                        ┌──────────────────────┐
                                        │ render               │
                                        │ Block → Layout       │
                                        │ (positioned []Prim)  │◄── tex (maths)
                                        └─────┬──────────┬─────┘◄── imgcache/youtube
                                              ▼          ▼
                                   ┌────────────────┐ ┌──────────────┐
                                   │ ui/paint.go    │ │ pdfout       │
                                   │ Prims → screen │ │ Prims → PDF  │
                                   └────────────────┘ └──────────────┘
```

The key idea: **`render` never draws and never knows about Gio or PDF.** It
only computes *what* goes *where* as a list of primitives (`Prim`). Two thin
painters turn those primitives into pixels (Gio) or PDF operators (fpdf).
That is why the preview and the PDF can't drift apart.

### Packages

| Package | Responsibility | Depends on |
|---|---|---|
| `cmd/mdedit` | `main`: flags, `-pdf` export, opens the window (desktop and browser) | ui, mdmodel, pdfout |
| `cmd/mdedit-web` | Static web server for the browser version (§8) | — |
| `internal/ui` | Everything you see and click: window, editor, preview painter, menus, toolbar, dialogs, find bar, keys | edit, mdmodel, render, pdfout, tex, imgcache, youtube |
| `internal/edit` | Formatting commands as **pure functions** on `State` (text + selection): bold, lists, headings, align, tables, TOC, search/replace, Enter/Tab | mdmodel (for TOC), youtube |
| `internal/mdmodel` | Parses Markdown into a flat list of styled `Block`s; maths syntax; syntax highlighting | goldmark, chroma, youtube |
| `internal/render` | Lays out a `Block` at a given width into a `Layout` of positioned `Prim`s | mdmodel, tex, imgcache, vfs |
| `internal/tex` | TeX maths typesetter; embeds the STIX Two Math font; glyph outlines | x/image/font/sfnt |
| `internal/pdfout` | Paints `Layout`s onto PDF pages, splitting blocks at page breaks | render, tex, imgcache, vfs, fpdf |
| `internal/imgcache` | Downloads YouTube stills into a cache (async for the preview, sync for PDF) | youtube, vfs |
| `internal/vfs` | File storage for images: the disk on the desktop, memory in the browser (§8) | — |
| `internal/youtube` | YouTube URLs and video-ID parsing | — |
| `internal/lint` | A test enforcing the code style (see §12) | — |
| `tools/mkicon` | Draws the application icon: hicolor PNGs, or a macOS iconset with `-macos` | x/image/vector |
| `tools/glyphdump` | Prints a character's glyph number, stored points and curve segments (§6) | go-text |
| `web/`, `assets/freebsd/` | The web page around the browser version; the FreeBSD rc.d script for `mdedit-web` (§8) | — |

Rule of thumb: **lower layers never import higher ones.** If you find
yourself wanting `render` to import `ui`, the code belongs somewhere else.

---

## 2. The data that flows through the layers

Four types carry almost everything. Know these and the rest follows.

### `edit.State` — text plus selection (edit/edit.go)

```go
type State struct {
    Text       []rune
    Start, End int // rune offsets; Start == End is just a caret
}
```

Every editing command is `func(edit.State) edit.State`. They don't touch the
UI, so they are trivially testable (see `edit/edit_test.go`, which writes
states as `"a [word] b"` for a selection or `"text|"` for a caret).

### `mdmodel.Block` and `Span` — the parsed document (mdmodel/model.go)

The document is a **flat** list of blocks. Containers (block quotes, list
items) are not nodes: each leaf block carries the stack of containers it is
in, in `Decor`. A flat list is cheap for the preview to virtualise and easy
for the PDF writer to split across pages.

```go
type Block struct {
    Kind  Kind      // Paragraph, Heading, Code, Rule, Table, Image, HTML, Math
    Level int       // heading level
    Spans []Span    // inline content (styled runs of text, inline maths)
    Decor []Decor   // enclosing quotes / list items, outermost first
    Table *TableData
    Src, Alt, Link string; Video bool // images and YouTube videos
    ID    string    // heading anchor, "#overview"
    Line  int       // 0-based source line (used to sync preview to caret)
    Align Align     // from <div align="…">
    Gap   float32   // space above, in em
    …
}

type Span struct {
    Text string
    Bold, Italic, Mono, Strike, Underline bool
    Math, Display bool  // Text holds TeX source
    Link  string
    Highlighted bool              // syntax-highlighted code …
    Token       chroma.TokenType  // … and its token type (keyword, string …)
}
```

Highlighted spans carry the *kind* of token, not a colour: the renderer
picks the colour from its theme (`render/inline.go: applyCodeStyle`), so one
parse serves both the light and the dark preview.

### `render.Layout` and `render.Prim` — what to draw where (render/render.go)

```go
type Layout struct {
    Prims  []Prim
    Height float32
    Breaks []float32 // y offsets where the PDF may split the block across pages
}

type Prim struct {
    Kind PrimKind // PRect, PEllipse, PLine, PText, PImage, PLink, PGlyph, PPoly
    X, Y, W, H float32 // PText/PGlyph: Y is the baseline
    Color color.NRGBA
    Text string; Font FontSpec; Size float32 // text
    URL string                               // link target or image source
    Face uint8; Glyph uint16; XScale, YScale float32 // maths glyph outline
    Points [][2]float32                      // polygon
    …
}
```

Coordinates: **y points down**, origin at the top-left of the block, units
are whatever `Ctx.Em` is in (pixels on screen, points in PDF).

### `render.Ctx` — the device the layout is for (render/render.go)

```go
type Ctx struct {
    Em            float32 // base font size in device units
    Measure       func(f FontSpec, size float32, s string) Metrics
    ImageSize     func(src string) (w, h int, ok bool)
    VideoControls bool    // preview only: play button + loading placeholder
    MaxImageHeight float32
    Theme         *Theme
}
```

The screen and the PDF each provide their own `Ctx`: the preview measures
text with Gio's shaper (`ui/textpaint.go`), the PDF with fpdf
(`pdfout/fonts.go: measure`). Everything else in `render` is shared.

---

## 3. The frame loop (Gio is immediate-mode)

Gio has no retained widget tree and no callbacks wired to buttons. **The
whole window is re-run as a function for every frame**:

```go
// ui/app.go: Run
for {
    switch e := w.Event().(type) {
    case app.DestroyEvent:
        a.saveRecovery()
        return e.Err
    case app.FrameEvent:
        gtx := app.NewContext(&ops, e)
        a.layout(gtx)      // handle input, then draw everything
        e.Frame(gtx.Ops)   // hand the drawing to the GPU
    }
}
```

```go
// ui/window.go: layout — the order of every frame
func (a *App) layout(gtx layout.Context) {
    a.handleInput(gtx)        // shortcuts, Enter/Tab/⌘-keys, toolbar, find bar
    a.handleEditorEvents(gtx) // typed text → reparse
    a.handleRightClick(gtx)
    a.updateTitle()
    size := gtx.Constraints.Max
    a.layoutWindow(gtx)       // menu bar, toolbar, panes, status bar
    a.restoreSelection()
    gtx.Constraints = layout.Exact(size)
    a.layoutOverlay(gtx)      // dialog, context menu or open drop-down
}
```

Two rules follow from this, and most input bugs come from forgetting one:

1. **Input waits in a queue until someone asks for it.** `gtx.Event(filter…)`,
   `clickable.Clicked(gtx)` and `editor.Update(gtx)` each *take* matching
   events out of the queue. **Whoever asks first wins.** That is how mdedit
   steals Enter, Tab and ⌘← from Gio's editor: `handleInput` runs *before*
   the editor sees the keys (`ui/keys.go: handleEnter`, `editorKey`).
2. **Drawing registers where input can arrive next frame.**
   `clickable.Layout(…)` both draws a button and marks its area clickable;
   `event.Op` / `gesture.Click.Add` do the same for custom areas. Something
   not drawn this frame can't be clicked next frame.

Files the operating system asks to open (the macOS Finder's "Open With",
or a file dropped on the Dock icon) arrive in `cmd/mdedit/main.go: opener`
on the main goroutine and are handed over with `ui/app.go: RequestOpen`;
`ui/window.go: handleOpenRequest` opens them on the next frame.

Anything that must happen *later* (after a dialog closes, after a download
finishes) is stored in a field and done on the next frame: see
`App.pendingEdit`, `App.pendingCmd` and `ui/window.go: runPending`, and
`Preview.imagesChanged` + `Preview.Invalidate`.

---

## 4. Walkthrough: clicking Format → Bold

1. **Frame N — the bar is drawn.** `ui/window.go: menuBarRow` →
   `ui/menubar.go: menuBar.Layout` → each title is `menuTitle.layout`, which
   wraps the label in `t.m.titles[t.i].Layout(gtx, t.body)` — a
   `widget.Clickable`.
2. **Click "Format" → frame N+1.** `menuBar.handleClicks` sees
   `m.titles[i].Clicked(gtx)` and sets `m.open`. Later in the frame,
   `ui/window.go: layoutOverlay` → `menuBar.Overlay` → `ui/popup.go:
   popup.Layout` draws the items, each wrapped in its own `Clickable`
   (`popupRow.layout`).
3. **Click "Bold" → frame N+2.** `popup.Layout` → `handleInput` →
   `clickedItem` returns the `*command`; `menuBar.Overlay` closes the menu
   and calls `chosen.run(gtx)`.
4. **The command.** Bold was built in `ui/commands.go: buildSharedCommands`:

   ```go
   c.bold = a.editCommand("Bold", "B", 0, inlineMarker(edit.Bold).apply)
   ```

   `ui/edits.go: editCommand` makes `run = editAction{a, f}.run`, which calls
   `applyEdit`:

   ```go
   func (a *App) applyEdit(gtx layout.Context, f editFunc) {
    a.setEditorState(f(a.editorState()))   // pure edit on text+selection
    gtx.Execute(key.FocusCmd{Tag: &a.editor})
   }
   ```

   `setEditorState` diffs old and new text and replaces only the changed
   part with `a.editor.Insert(…)`, so the change is **one undo step**, then
   calls `a.reparse()` — from here it is the same as typing (§5).

The keyboard shortcut ⌘B/Ctrl+B reaches the same `run`:
`ui/window.go: handleInput` → `ui/keys.go: handleShortcuts` → `runShortcut`.
The toolbar button reaches it via `ui/window.go: handleToolbar`.

---

## 5. Walkthrough: a keystroke to pixels in the preview

1. **The key arrives** for the focused widget (`&a.editor`) and Gio sends a
   frame.
2. **Special keys are intercepted** in `ui/window.go: handleInput`:
   `handleEnter` (list continuation, `edit.Newline`), `handleTab`
   (`edit.Tab`/`edit.Outdent`), `handleMacKeys` (⌘-arrows). Ordinary
   characters don't match their filters and stay in the queue.
3. **The editor takes the key.** `ui/window.go: handleEditorEvents` calls
   `a.editor.Update(gtx)`; Gio's `widget.Editor` inserts the character and
   reports a `ChangeEvent` → `editorChanged` → `a.reparse()`.
4. **Re-parse** (`ui/app.go: reparse`):

   ```go
   a.version++                        // find bar: search again
   a.doc = mdmodel.Parse([]byte(txt)) // Markdown → []Block
   a.preview.SetDocument(a.doc)       // drops the preview's layout cache
   ```

5. **Draw, same frame.** `layoutWindow` → `body` → `ui/panes.go: panes` →
   `splitPanes`, which lays out `editorPane` on the left and `previewPane`
   on the right. `followCaret` scrolls the preview to the block holding the
   caret (`Block.Line` → `Document.BlockAtLine`).
6. **Preview** (`ui/preview.go`): `Preview.Layout` → `measure` (content
   width; resets the cache if width, font size or images changed) →
   `material.List(…).Layout(…, p.listItem)`. The list is **virtualised**:
   only visible blocks are laid out. `block` → `blockLayout` (cache, else
   `p.ctx.Block(…)`) → `p.paint(…)`.
7. **Layout** (`render/block.go: Ctx.Block`) → `frame` (indentation, quote
   bars, list markers from `Decor`) → `content` (switch on `Block.Kind`) →
   e.g. `render/render.go: paragraph` → `render/inline.go: layoutInline`
   (line breaking) and `emitLines` (→ `Prim`s).
8. **Paint** (`ui/paint.go: paint` → `paintPrim`, switch on `Prim.Kind`) →
   e.g. text via `ui/textpaint.go: textPainter.draw` (Gio's shaper fills the
   glyph outlines). Back in `Run`, `e.Frame` sends it to the GPU. §6 follows
   one character from here down to the pixels.

The PDF export takes the same path from step 7 on, with
`pdfout/pdf.go: Write` → `placeBlock` → `Ctx.Block` →
`pdfout/paging.go: place` (page splitting at `Layout.Breaks`) →
`pdfout/draw.go: drawPrim`.

---

## 6. From a character to pixels

§5 ends with "Gio's shaper fills the glyph outlines". This section opens
that box: how the letter **o** in a paragraph becomes coloured pixels. Most
of this happens in libraries, not in mdedit, so the references below name
the module as well as the file:

| Short name | Module (see `go.mod` for versions) |
|---|---|
| go-text | `github.com/go-text/typesetting` — font parsing and shaping, used by Gio |
| Gio | `gioui.org` — text shaper, paths, GPU renderer |
| Gio shaders | `gioui.org/shader`, directory `gio/` — the GPU programs |
| sfnt | `golang.org/x/image/font/sfnt` — font parsing, used by mdedit's maths |

The short version: **a character's shape is not computed, it is looked
up.** A type designer drew every glyph as curves; the font file stores
them as points. Drawing text means finding the right glyph, decoding its
points into curves, scaling them to the text size, and letting the GPU work
out how much of each pixel lies inside the curves.

```
'o'  ──cmap──▶ glyph 82 ──loca──▶ bytes 11340–11468 ──glyf──▶ 20 points
     ──implied midpoints──▶ 14 curve segments (font units, y up)
     ──× size/2048, flip y──▶ clip path (pixels, y down)
     ──GPU pass 1──▶ coverage per pixel (0 … 1)
     ──GPU pass 2──▶ colour × coverage, blended over the background
```

Every number in this section comes from the real Go font, and you can print
them yourself for any character:

    go run ./tools/glyphdump o

### 6.1 The font files

The preview draws text with the **Go fonts** (regular, bold, italic, bold
italic, and the same four in Go Mono). They are TrueType files compiled
into the binary; `ui/app.go: New` hands them to Gio's shaper:

```go
shaper := text.NewShaper(text.WithCollection(gofont.Collection()))
```

`ui/textpaint.go: giofont` picks one of the eight from a `render.FontSpec`
(`Mono`, `Bold`, `Italic`). Maths, and characters the Go fonts lack (⌘, ⇧
…), come from **STIX Two Math**, embedded in `internal/tex/fonts/` (§6.9).

A TrueType file is a directory of tables, each named by a four-letter tag.
Four of them matter for drawing:

| Table | Holds |
|---|---|
| `head` | Units per em (2048 for the Go fonts) and the format of `loca` |
| `cmap` | Character → glyph number |
| `loca` | Glyph number → where its outline starts in `glyf` |
| `glyf` | The outlines: points, flags, contours |
| `hmtx` | Each glyph's advance width (how far the pen moves after it) |

Coordinates in a font are integers in **font units**: 2048 units = 1 em =
the font size, the baseline is y = 0, and **y points up** (the opposite of
the screen).

### 6.2 Step 1 — character to glyph number (`cmap`)

Fonts don't index glyphs by character: one character may have several
glyphs (ligatures, alternates) and some glyphs belong to no character.
The `cmap` table maps characters to glyph numbers. For characters up to
U+FFFF it is a sorted list of ranges, searched by bisection
(go-text `font/cmap.go: cmap4.Lookup`):

```go
c := uint16(r)
for i, j := 0, len(s); i < j; {
    h := i + (j-i)/2
    entry := s[h]
    if c < entry.start {
        j = h
    } else if entry.end < c {
        i = h + 1
    } else if entry.indexes == nil {
        return GID(c + entry.delta), true        // whole range: glyph = char + offset
    } else {
        glyph := entry.indexes[c-entry.start]    // or one entry per character
        …
        return GID(uint16(glyph) + entry.delta), true
    }
}
return 0, false
```

Gio's shaper calls this for every character (go-text
`harfbuzz/font.go: nominalGlyph`). **'o' (U+006F) → glyph 82.**

A character that isn't in the font gets **glyph 0**, the "missing glyph"
box. That is why ⌘ once showed as a square in exported PDFs, and why
`pdfout/fonts.go: needsFallback` now checks each character first.

### 6.3 Step 2 — shaping: which glyphs, and where

Looking up glyphs one by one isn't enough to set a line of text: some
pairs are moved closer together (kerning, `GPOS` table), some sequences
become one glyph (ligatures, `GSUB` table), and right-to-left scripts are
reordered. Gio runs go-text's port of HarfBuzz for this. mdedit calls it in
`ui/textpaint.go: textPainter.shape`:

```go
t.shaper.LayoutString(text.Parameters{
    Font:     giofont(f),
    PxPerEm:  fixed.Int26_6(size * 64), // 26.6 fixed point: 1/64 pixel
    MaxWidth: 1 << 24,                  // never wrap: render has already
}, s)                                   // broken the lines
for g, ok := t.shaper.NextGlyph(); ok; g, ok = t.shaper.NextGlyph() {
    t.glyphs = append(t.glyphs, g)
}
```

Each `text.Glyph` has a packed **ID** (glyph number, font index and pixel
size; Gio `text/shaper.go: splitGlyphID` unpacks it), its pen position
`X`, its `Advance` (from `hmtx`, adjusted by kerning), and the line's
`Ascent` and `Descent`.

The same function serves **measuring**: `textPainter.measure` adds up the
advances and caches the result. `render` measures with it when breaking
lines, so the widths it plans with are exactly the widths that get painted.

### 6.4 Step 3 — glyph number to bytes (`loca`)

`loca` is an array of `numGlyphs + 1` offsets into `glyf`; glyph *n* is
the bytes from `loca[n]` to `loca[n+1]` (go-text
`font/opentype/tables/glyphs_glyf_src.go: ParseLoca`, `ParseGlyf`):

```go
for i := range out {
    out[i] = 2 * uint32(binary.BigEndian.Uint16(src[2*i:])) // short format stores offset/2
}
…
start, end := locaOffsets[i], locaOffsets[i+1]
if start == end {   // no outline, e.g. the space
    continue
}
out[i], _, err = ParseGlyph(src[start:end])
```

**Glyph 82 is bytes 11340–11468 of `glyf`** — the whole letter is 128
bytes.

### 6.5 Step 4 — bytes to points (`glyf`)

A simple glyph is stored as: the bounding box; the index of the **last
point of each contour** (a contour is one closed loop); hinting
instructions; one **flag byte per point**; then all x coordinates, then all
y coordinates. To save space each coordinate is a **difference from the
previous point**, in 0, 1 or 2 bytes as the flag says:

| Flag bit | Meaning |
|---|---|
| `0x01` | The point is **on** the curve (else it is a control point) |
| `0x02` / `0x04` | The x / y difference is 1 byte |
| `0x10` / `0x20` | With a 1-byte difference: it is positive. With 2 bytes: there is none, the coordinate repeats |
| `0x08` | The next byte says how many more points use this same flag |

go-text `font/opentype/tables/glyphs_glyf_src.go`:

```go
func readContourPoint(flag byte, data []byte, pos int, shortFlag, sameFlag uint8) (int, int16) {
    var v int16
    if flag&shortFlag != 0 {         // 1 byte, sign in the flag
        val := data[pos]
        pos++
        if flag&sameFlag != 0 {
            v += int16(val)
        } else {
            v -= int16(val)
        }
    } else if flag&sameFlag == 0 {   // 2 bytes, signed
        val := binary.BigEndian.Uint16(data[pos:])
        pos += 2
        v += int16(val)
    }                                // else 0 bytes: unchanged
    return pos, v
}

func parseGlyphContourPoints(dataX, dataY []byte, points []GlyphContourPoint) {
    …
    for i, p := range points {
        posX, offsetX = readContourPoint(p.Flag, dataX, posX, xShortVector, xIsSameOrPositiveXShortVector)
        vX += offsetX                // running total = absolute coordinate
        points[i].X = vX
        …
    }
}
```

go-text `font/glyphs.go: getContourPoints` then keeps, per point, its
position, whether it is on the curve, and whether it ends a contour.

The **'o'**:

```
contours end at points [11 19]       → points 0–11: outer ring; 12–19: the hole
  point  0  flag 0x05  (  562,   -25)  on-curve       0x05 = on + y is 1 byte, negative: 0 − 25
  point  1  flag 0x22  (  343,   -25)  off-curve      0x22 = x is 1 byte, negative: 562 − 219; y repeats
  point  2  flag 0x00  (   86,   283)  off-curve      2-byte x and y
  point  3  flag 0x11  (   86,   543)  on-curve       leftmost point
  point  4  flag 0x10  (   86,   806)  off-curve
  point  5  flag 0x00  (  345,  1110)  off-curve
  point  6  flag 0x33  (  569,  1110)  on-curve       top
  point  7  flag 0x32  (  792,  1110)  off-curve
  point  8  flag 0x00  ( 1052,   807)  off-curve
  point  9  flag 0x11  ( 1052,   545)  on-curve       rightmost point
  point 10  flag 0x10  ( 1052,   277)  off-curve
  point 11  flag 0x00  (  792,   -25)  off-curve
  point 12  flag 0x27  (  565,   123)  on-curve       ← the hole
  point 13  flag 0x20  (  839,   123)  off-curve
  point 14  flag 0x11  (  839,   548)  on-curve
  point 15  flag 0x10  (  839,   962)  off-curve
  point 16  flag 0x21  (  569,   962)  on-curve
  point 17  flag 0x20  (  299,   962)  off-curve
  point 18  flag 0x11  (  299,   543)  on-curve
  point 19  flag 0x10  (  299,   123)  off-curve
```

Two things to notice. The bottom is at y = −25 and the top at 1110, a
little beyond the baseline and the flat letters' height: round letters
**overshoot** on purpose, or they would look smaller than an "x". And the
hinting instructions — a small bytecode program that nudges points onto
the pixel grid at small sizes — are parsed but **never run**: neither
go-text nor Gio executes them. Gio relies on antialiasing (§6.8) instead.

### 6.6 Step 5 — points to curves (implied midpoints)

TrueType outlines are made of straight lines and **quadratic Bézier
curves**. A quadratic curve runs from one on-curve point to the next and is
pulled towards one off-curve **control point** between them:

    B(t) = (1−t)²·P₀ + 2(1−t)t·C + t²·P₁,   t from 0 to 1

To save space, TrueType has a shortcut: **two control points in a row
imply an on-curve point exactly halfway between them.** go-text
`font/renderer.go: buildSegments` restores those points while turning the
point list into segments:

```go
for _, point := range points {
    p := point.SegmentPoint
    if !firstOnCurveValid {
        if point.isOnCurve {            // first point of a contour
            firstOnCurve = p
            firstOnCurveValid = true
            out = append(out, Segment{Op: ot.SegmentOpMoveTo, Args: [3]SegmentPoint{p}})
        }
        …                               // (a contour may also start off-curve)
    } else if !lastOffCurveValid {
        if !point.isOnCurve {           // a control point: keep it, wait
            lastOffCurve = p
            lastOffCurveValid = true
            …
        } else {                        // on → on: a straight line
            out = append(out, Segment{Op: ot.SegmentOpLineTo, Args: [3]SegmentPoint{p}})
        }
    } else {
        if !point.isOnCurve {           // control → control: the implied point
            out = append(out, Segment{
                Op:   ot.SegmentOpQuadTo,
                Args: [3]SegmentPoint{lastOffCurve, midPoint(lastOffCurve, p)},
            })
            lastOffCurve = p
            lastOffCurveValid = true
        } else {                        // control → on: an ordinary curve
            out = append(out, Segment{Op: ot.SegmentOpQuadTo, Args: [3]SegmentPoint{lastOffCurve, p}})
            lastOffCurveValid = false
        }
    }
    if point.isEndPoint {
        …                               // a last line or curve back to the first point
    }
}
```

The **'o'** becomes 14 segments (each `QuadTo` lists its control point,
then its end point):

```
  MoveTo ( 562.0,  -25.0)                       point 0
  QuadTo ( 343.0,  -25.0) ( 214.5,  129.0)      control: point 1; end: implied, halfway to point 2
  QuadTo (  86.0,  283.0) (  86.0,  543.0)      control: point 2; end: point 3
  QuadTo (  86.0,  806.0) ( 215.5,  958.0)
  QuadTo ( 345.0, 1110.0) ( 569.0, 1110.0)
  QuadTo ( 792.0, 1110.0) ( 922.0,  958.5)
  QuadTo (1052.0,  807.0) (1052.0,  545.0)
  QuadTo (1052.0,  277.0) ( 922.0,  126.0)
  QuadTo ( 792.0,  -25.0) ( 562.0,  -25.0)      back to the start: outer ring closed
  MoveTo ( 565.0,  123.0)                       the hole
  QuadTo ( 839.0,  123.0) ( 839.0,  548.0)
  QuadTo ( 839.0,  962.0) ( 569.0,  962.0)
  QuadTo ( 299.0,  962.0) ( 299.0,  543.0)
  QuadTo ( 299.0,  123.0) ( 565.0,  123.0)
```

Check the implied point: ((343 + 86) / 2, (−25 + 283) / 2) = (214.5, 129).

Also look at the **directions**. The outer ring goes from the bottom to
the left, up and round to the right: clockwise. The hole goes from the
bottom to the right, up and round to the left: anticlockwise. That is not
a detail: it is the only thing that makes the hole a hole (§6.8). Letters
made of straight strokes simply use `LineTo`; try
`go run ./tools/glyphdump l`.

Other kinds of glyph take other routes to the same segment list:

- **Composite glyphs** are built from other glyphs, each moved or scaled
  (a font may store "é" as "e" plus "´"). go-text
  `font/glyphs.go: getPointsForGlyphRec` loads the parts recursively and
  applies their transforms. (The Go fonts happen to store "é" as a simple
  glyph with three contours.)
- **OpenType/CFF fonts** store **cubic** Bézier curves (two control
  points) in a `CFF` table instead of `glyf`
  (go-text `font/renderer.go: glyphDataFromCFF1`); they come out as
  `CubeTo` segments.
- **Colour emoji** are stored as bitmaps or SVG, not outlines. Gio draws
  those with `Shaper.Bitmaps` as images, which `textPainter.draw` adds
  after the outlines.

### 6.7 Step 6 — curves to a path in pixels

So far everything is in font units, y up. Gio `text/gotext.go:
shaperImpl.Shape` turns the run's glyphs into **one** `clip.Path` in
pixels: for each glyph it moves the pen to the glyph's `X` from shaping,
scales every point by `size / unitsPerEm`, and flips y:

```go
scaleFactor := fixedToFloat(ppem) / float32(face.Upem())   // e.g. 16 px / 2048
glyphData := face.GlyphData(gid)                            // steps 3–5
…
for _, fseg := range outline.Segments {
    …
        a := f32.Point{
            X: fseg.Args[i].X * scaleFactor,
            Y: -fseg.Args[i].Y * scaleFactor,               // font y up → screen y down
        }
    …
    switch fseg.Op {
    case gotextot.SegmentOpMoveTo:
        builder.Move(args[0])
    case gotextot.SegmentOpLineTo:
        builder.Line(args[0])
    case gotextot.SegmentOpQuadTo:
        builder.Quad(args[0], args[1])
    case gotextot.SegmentOpCubeTo:
        builder.Cube(args[0], args[1], args[2])
    }
}
```

At 16 px (one em = 16 px), the 'o''s rightmost point (1052, 545) becomes
(8.22, −4.26): 8.22 px right of the pen and 4.26 px above the baseline.
Coordinates stay fractional; nothing is snapped to whole pixels.

mdedit's `ui/textpaint.go: textPainter.draw` uses that path as a **clip**
and paints through it:

```go
st := op.Offset(image.Pt(rnd(x)+gs[0].X.Floor(), rnd(baseline))).Push(ops) // origin: start of the run, on the baseline
outline := clip.Outline{Path: t.shaper.Shape(gs)}.Op().Push(ops)          // only inside the letters …
paint.ColorOp{Color: c}.Add(ops)                                          // … with this colour …
paint.PaintOp{}.Add(ops)                                                  // … fill.
outline.Pop()
if call := t.shaper.Bitmaps(gs); call != (op.CallOp{}) {                  // colour emoji, if any
    call.Add(ops)
}
st.Pop()
```

None of this draws yet. These are operations recorded in `gtx.Ops`; at
the end of the frame (`e.Frame` in `ui/app.go: Run`) Gio hands the whole
list to its GPU renderer.

### 6.8 Step 7 — the GPU: coverage, then colour

A pixel is a small square, and a curve can cut through it. The renderer's
job is to find, for every pixel, **what fraction of that square lies
inside the outline** — its **coverage**, 0 to 1 — and mix the text colour
with the background in that proportion. Gio's GPU renderer (Gio
`gpu/path.go`) does it in two passes. On macOS the shaders run as Metal,
on Linux and FreeBSD as OpenGL or Vulkan; over XRDP or in a VM that is
often Mesa's software OpenGL, which runs the same programs on the CPU.

**Preparation.** The GPU program handles quadratic curves only, so each
`CubeTo` is first split into a few quadratics that follow it closely
(Gio `internal/stroke/stroke.go: SplitCubic`). Lines are quadratics with
the control point in the middle.

**Pass 1: coverage (Gio shaders `stencil.vert`, `stencil.frag`).** For
each curve segment the GPU draws a rectangle that reaches from the curve
down to the bottom of the whole shape (`stencil.vert`: "North" is
`maxy + 1`). For every pixel in that rectangle, `stencil.frag` runs:

```glsl
vec2 extent = clamp(vec2(vFrom.x, vTo.x), -0.5, 0.5);  // the part of this pixel's width the curve spans
…
float t = x0/(p1.x+sqrt(p1.x*p1.x+(v.x-p1.x)*x0));      // solve the Bézier for this pixel's x
float y = mix(mix(left.y, vCtrl.y, t), mix(vCtrl.y, right.y, t), t);  // the curve's height there
vec2 d_half = mix(p1, v, t);
float dy = d_half.y/d_half.x;                           // and its slope: within one pixel,
…                                                       // the curve is treated as a straight line
float width = extent.y - extent.x;                      // negative if the curve runs right to left
…
float area = 0.5*(sides.z - sides.z*sides.y + 1.0 - sides.x+sides.x*sides.w);
area *= width;                                          // the part of the pixel below the line, signed
fragCover.r = area;
```

So each segment gives every pixel below it a number: how much of the
pixel lies below the curve (1 for pixels well below it, a fraction for
pixels it cuts through), **positive or negative depending on the
direction the segment runs**. The results are **added** into an offscreen
coverage texture (`gpu/path.go`: blend factors `One`, `One`).

Adding is what makes the shape. Follow one column of pixels down through
the middle of the 'o':

```
                                     top of outer ring (runs →)   adds +1 below it
  pixel above the 'o'          0
  pixel in the ring            +1                                 = 1   inside
                                     top of the hole (runs ←)     adds −1 below it
  pixel in the hole            +1 −1                              = 0   outside
                                     bottom of the hole (runs →)  adds +1 below it
  pixel in the ring            +1 −1 +1                           = 1   inside
                                     bottom of outer ring (runs ←) adds −1 below it
  pixel below the 'o'          +1 −1 +1 −1                        = 0   outside
```

Pixels that a curve cuts through get fractions — 0.3, 0.6, 0.85 — and
those fractions are the **antialiasing**. The same arithmetic works for
any shape with any number of contours: every glyph in every font, maths
symbols, rounded rectangles. That is why fonts must wind holes in the
opposite direction to their outlines.

**Pass 2: colour (Gio shader `cover.frag`).** This is mdedit's
`ColorOp` + `PaintOp`. For each pixel of the shape:

```glsl
fragColor = {{.FetchColorExpr}};                      // the paint: colour, image or gradient
float c = min(abs(texture(cover, vCoverUV).r), 1.0);  // this pixel's coverage from pass 1
fragColor *= c;
```

and the result is blended over the window so far (premultiplied alpha):

    pixel = textColour × c + background × (1 − c)

One edge pixel of dark-theme text (`#e6edf3`) on the dark background
(`#0d1117`), with `c = 0.25`:

```
R = 0xe6 × 0.25 + 0x0d × 0.75 ≈ 67  = 0x43
G = 0xed × 0.25 + 0x11 × 0.75 ≈ 72  = 0x48
B = 0xf3 × 0.25 + 0x17 × 0.75 ≈ 78  = 0x4e      → #43484e
```

Pixels with `c = 1` get exactly `#e6edf3`, pixels in the hole stay
`#0d1117`.

Everything else the preview draws goes through the same two passes:
`paint.FillShape` in `ui/paint.go` is clip + `ColorOp` + `PaintOp` for a
rounded rectangle, an ellipse, a stroked line or a polygon instead of a
glyph.

### 6.9 Maths glyphs: mdedit does steps 1–6 itself

Formulas are laid out by mdedit's own TeX engine, not by Gio's shaper, so
mdedit looks up and decodes the STIX glyphs itself, with sfnt.
`tex/font.go`:

```go
// lookup: character → (font, glyph number), via each face's cmap
for i, fc := range faces {
    if g, err := fc.f.GlyphIndex(&fc.buf, r); err == nil && g != 0 {
        return uint8(i), uint16(g), true
    }
}

// Outline: glyph number → segments in em, cached
segs, err := fc.f.LoadGlyph(&fc.buf, sfnt.GlyphIndex(g), fc.upem, nil) // loca + glyf + implied points
…
out = toSegs(segs, float32(fc.upem))                                      // ÷ units per em → em
```

sfnt already returns y pointing down. Tall delimiters are not single
characters: `tex/delims.go` builds them from the Unicode bracket pieces
(⎛ ⎜ ⎝ for a big "(") or stretches one glyph with `Item.YScale`; wide
accents stretch with `XScale`. Both end up in `Prim.XScale`/`YScale`.

`ui/paint.go: paintGlyph` is step 6 by hand — scale the em coordinates
to pixels and build the `clip.Path`:

```go
segs := tex.Outline(pr.Face, pr.Glyph)
t := glyphTransform{x: x, y: pr.Y, sx: pr.Size * pr.XScale, sy: pr.Size * pr.YScale}
var path clip.Path
path.Begin(gtx.Ops)
for i, s := range segs {
    addSegment(&path, s, t, i > 0) // MoveTo/LineTo/QuadTo/CubeTo; a MoveTo closes the previous contour
}
path.Close()
paint.FillShape(gtx.Ops, pr.Color, clip.Outline{Path: path.End()}.Op())
```

From there it is step 7 as for any text.

### 6.10 Images: the same pipeline, a different paint

Glyphs are paths; an image is **not** — not even a path per pixel. Every
paint operation answers two separate questions, and glyphs and images sit
at opposite ends of them:

| | **Where** to paint: the clip | **What colour** each pixel gets: the paint |
|---|---|---|
| Glyph | Complicated: the outline's curves | Simple: one colour, `paint.ColorOp` |
| Image | Simple: one rectangle, four corners | Complicated: a grid of colours, `paint.ImageOp` |
| Gradient | Any shape | A colour computed from the position, `paint.LinearGradientOp` |

Side by side, the two painters differ in exactly those two lines:

```go
// text — ui/textpaint.go: textPainter.draw
outline := clip.Outline{Path: t.shaper.Shape(gs)}.Op().Push(ops) // where: the letters' curves
paint.ColorOp{Color: c}.Add(ops)                                 // what: one colour
paint.PaintOp{}.Add(ops)

// image — ui/paint.go: paintImage
img.op.Add(gtx.Ops)                          // what: the image's pixels
cl := clip.Rect{Max: img.size}.Push(gtx.Ops) // where: a plain rectangle
paint.PaintOp{}.Add(gtx.Ops)
```

#### What an image is

An image is **raster** data: a grid of width × height samples ("texels"
once on the GPU), each four bytes — red, green, blue, alpha. Nothing in it
describes shape, only colours at fixed positions. `ui/images.go:
decodeImage` undoes the PNG/JPEG/GIF compression into that grid and wraps
it for Gio:

```go
img, _, err := image.Decode(f)                         // compressed file → pixel grid
…
return &imageEntry{op: paint.NewImageOp(img), size: img.Bounds().Size()}
```

A 1280 × 720 YouTube still is 921,600 samples, about 3.7 MB once
decoded (the JPEG on disk is a few hundred KB). Where the image comes from
and how big it is drawn is §7; here it arrives as
`Prim{Kind: PImage, X, Y, W, H, URL}` — the box to fill, in device units.

#### Step A — uploaded once as a texture

The first time an `ImageOp` is drawn, Gio copies the grid into GPU memory
as a **texture** and keeps it in a cache for as long as the same op is
used (Gio `gpu/gpu.go`, `renderer.texHandle`):

```go
var minFilter, magFilter driver.TextureFilter
switch data.filter {
case filterLinear:                 // the default for paint.ImageOp
    minFilter, magFilter = driver.FilterLinearMipmapLinear, driver.FilterLinear
case filterNearest:
    minFilter, magFilter = driver.FilterNearest, driver.FilterNearest
}
handle, err := r.ctx.NewTexture(driver.TextureFormatSRGBA,
    data.src.Bounds().Dx(), data.src.Bounds().Dy(),
    minFilter, magFilter,
    driver.BufferBindingTexture,
)
…
driver.UploadImage(handle, image.Pt(0, 0), data.src)
```

`FilterLinearMipmapLinear` makes the driver also build **mipmaps**:
copies of the image halved again and again — 640 × 360, 320 × 180 … down to
1 × 1 — used when the image is drawn smaller than it is (Gio
`gpu/internal/opengl/opengl.go`: `GenerateMipmap`; on macOS
`gpu/internal/metal/metal_darwin.go`: `blitEncGenerateMipmapsForTexture`).

This is why `ui/images.go` caches the `imageEntry` and its `ImageOp`: a
new `ImageOp` for the same picture every frame would mean decoding and
uploading 3.7 MB every frame.

#### Step B — a rectangle, placed by a transform

`ui/paint.go: paintImage`:

```go
func (p *Preview) paintImage(gtx layout.Context, pr *render.Prim, x float32) {
    img := p.image(pr.URL)
    if img == nil {
        return
    }
    scale := f32.Pt(pr.W/float32(img.size.X), pr.H/float32(img.size.Y))      // image pixels → box size
    at := f32.Pt(float32(rnd(x)), float32(rnd(pr.Y)))                        // box position, whole pixels
    st := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, scale).Offset(at)).Push(gtx.Ops)
    img.op.Add(gtx.Ops)
    cl := clip.Rect{Max: img.size}.Push(gtx.Ops)
    paint.PaintOp{}.Add(gtx.Ops)
    cl.Pop()
    st.Pop()
}
```

The clip is the image's own rectangle **in image pixels**, `(0,0)–(1280,
720)`. The affine transform (scale, then move) maps that rectangle onto
the box from the layout, here `(0,0)–(528, 297)`. Because the paint and
the clip are transformed together, the image always fills exactly its
rectangle, whatever the scale. A rectangle has no curves, so it needs no
coverage pass (§6.8): every screen pixel inside it has coverage 1.

#### Step C — every screen pixel looks up its colour

The colour pass is the same `cover.frag` as for text (§6.8). Its first
line is a template, filled in per kind of paint when Gio's shaders are
generated (Gio shaders `cmd/convertshaders/main.go`):

```go
FetchColorExpr: `_color.color`,                                                    // ColorOp: one colour
FetchColorExpr: `mix(_gradient.color1, _gradient.color2, clamp(vUV.x, 0.0, 1.0))`, // gradient
FetchColorExpr: `texture(tex, vUV)`,                                               // ImageOp: look it up
```

`vUV` is the pixel's position inside the rectangle, 0 to 1 across and
down. `texture()` is answered by the GPU's **texture unit**, hardware made
for this: it maps (u, v) into the image and, with linear filtering,
**blends the nearest samples by distance** rather than picking one.

A worked pixel: the 1280 × 720 still drawn at 528 × 297 (§7.5), screen
pixel (100, 50) measured from the image's corner, sampled at its centre:

```
in the box:     u = 100.5 / 528 ≈ 0.190     v = 50.5 / 297 ≈ 0.170
in the image:   0.190 × 1280 ≈ 243.6        0.170 × 720 ≈ 122.4
```

That point falls *between* image pixels, and each screen pixel covers
about 1280 / 528 ≈ 2.4 image pixels across. So the texture unit:

1. picks the two mipmap levels around a 2.4× reduction: 640 × 360 (2×)
   and 320 × 180 (4×);
2. in each, blends the 4 samples around the point, weighted by how close
   they are (bilinear);
3. mixes the two results by where 2.4 lies between 2 and 4 (trilinear).

The screen pixel thus gets roughly the **average of the 2.4 × 2.4 patch
of the original** it covers. Without mipmaps, shrinking would sample
single pixels and skip the ones between: fine detail turns into jagged
edges, and moiré patterns that shimmer while scrolling.

The result is multiplied by the coverage (1) and blended over the
background, exactly as for text.

#### Paths or pixels: why both exist

You *could* draw an image as 921,600 one-pixel squares with a `ColorOp`
each; it would be enormously slower and look worse when scaled, each
square getting a hard edge instead of a blend. And you could store a
glyph as a picture; it would go soft or blocky at every size but one.
The two describe different things:

| | Path (vector) | Texture (raster) |
|---|---|---|
| Stores | Shape: points and curves | Colour: a fixed grid of samples |
| Enlarged | Stays sharp — the outline is recomputed at the new size | Goes soft — the GPU blends between samples it doesn't have |
| Shrunk | Stays sharp | Fine, with mipmaps |
| Good for | Text, maths, rules, borders, icons | Photos, screenshots, video stills |

mdedit uses each where it fits. The YouTube still is a texture, but the
**play button** on top of it is a red rounded rectangle and a white
triangle (`render/image.go: playButton` → `PRect` + `PPoly`), and a
ticked **task box** is a filled rounded rectangle with two stroked lines
for the tick (`render/render.go: marker` → `PRect` + 2 × `PLine`), so both
stay sharp at any size, on any screen, and in the PDF.

### 6.11 The PDF: the viewer rasterises

The PDF export never produces pixels. `pdfout` writes **instructions** and
the PDF viewer (or printer) does steps 5–7 at whatever resolution it
needs, which is why PDF text stays sharp at any zoom.

- **Text** (`pdfout/fonts.go: drawText`): the eight Go fonts are embedded
  in the PDF (`registerFonts`; fpdf includes only the glyphs used), and
  each run becomes a text instruction: font, size, position, string. The
  viewer then does its own `cmap` → `glyf` → curves → coverage.
- **Fallback characters** (⌘, ⇧, ✓ …) and **all maths**: mdedit decodes
  the outline itself (`tex.Outline`, as in §6.9) and writes it as a
  filled path, `pdfout/fonts.go: fillGlyph` → `pathSegment` (`MoveTo`,
  `LineTo`, `CurveTo`; fill rule nonzero, the same idea as §6.8).
- **Images** are embedded as files (PNG, JPEG, GIF) and placed with a
  size; the viewer scales them (§7.7).

Measuring for the PDF uses fpdf's widths (`pdfout/fonts.go: measure`),
taken from the same `hmtx` advances, so lines break at the same words as
on screen.

---

## 7. Images and YouTube videos

Images and videos take the same road as text — Markdown → `Block` →
`Layout` → painter — with two extra problems: the pixels live **somewhere
else** (a file, or YouTube's servers), and they may **arrive late**. This
section follows both kinds from the Markdown to the screen and the PDF.

```
![Alt](src)                       ┐
[![Title](still URL)](watch URL)  ┘ Markdown
      │ mdmodel/parse.go: paragraph, soleImage
      ▼
Block{Kind: Image, Src, Alt, Link, Video}
      │ render/images.go: ResolveImage ── remote YouTube still? ──▶ imgcache (download, disk cache)
      │ Ctx.ImageSize (preview: decode / PDF: file header)
      ▼ render/image.go: Ctx.image, fitImage, playButton
Layout{PImage | placeholder PRect, play button PRect + PPoly, PLink}
      │
      ├─▶ ui/paint.go: paintImage (texture, §6.10)       click → followLink → browser
      └─▶ pdfout/draw.go: drawImage (embedded file)      click → PDF link annotation
```

### 7.1 How they are written

mdedit has no video syntax of its own; it uses the notation GitHub
READMEs use, so documents look the same on GitHub:

```markdown
![A diagram](images/diagram.png)                          an image

[![Logo](logo.png)](https://example.com)                  an image that is a link

[![Video Title](https://img.youtube.com/vi/dQw4w9WgXcQ/maxresdefault.jpg)](https://www.youtube.com/watch?v=dQw4w9WgXcQ)
                                                          a video: a link to the watch page
                                                          around the video's still image
```

A video is therefore **just a linked image** whose link target is a
YouTube video. The still (`maxresdefault.jpg`) is the picture YouTube shows
before you press play; GitHub shows exactly this, and so does mdedit.
Nothing is ever played inside mdedit: a click opens the browser.

An image is only drawn as a picture when it is **alone in its paragraph**
(possibly wrapped in one link). An image in the middle of a sentence is
shown as its alt text, `[image: Alt]`, in italics and clickable
(`mdmodel/inline.go: inlineRun.image`); drawing pictures inside lines of
text would complicate line breaking for little gain.

### 7.2 Inserting a video

Insert → YouTube Video… (or the toolbar button) opens
`ui/videodialog.go: VideoDialog`, which asks for a video ID and a title.
The ID field also accepts a pasted link; `youtube.ParseID`
(`internal/youtube/youtube.go`) understands the forms in use:

```go
func ParseID(s string) (string, bool) {
    s = strings.TrimSpace(s)
    if ValidID(s) {                 // 11 characters of [A-Za-z0-9_-]
        return s, true
    }
    u, err := url.Parse(s)
    if err != nil || !isYouTubeHost(u.Hostname()) {
        return "", false
    }
    id := idFromURL(u)              // ?v=ID, youtu.be/ID, /embed/ID, /shorts/ID, /live/ID, /vi/ID
    return id, ValidID(id)
}
```

The dialog reports to `ui/commands.go: videoChosen`, which can't edit the
text itself (dialogs have no `gtx`), so it leaves the edit for the next
frame:

```go
func (a *App) videoChosen(id, title string) {
    a.dialog = nil
    if id != "" {
        a.pendingEdit = videoLink{id, title}.apply  // ui/edits.go
    }
}
```

`videoLink.apply` calls `edit.Video`, which puts the Markdown from
`youtube.Markdown(id, title)` on a line of its own (`edit/edit.go:
ownLine`). From there it is ordinary typing: `reparse`, and §5.

### 7.3 Parsing: an image block

goldmark parses `![Alt](src)` as an `ast.Image` inside a paragraph, and a
linked image as an `ast.Image` inside an `ast.Link`. `mdmodel/parse.go:
paragraph` checks whether that is all the paragraph contains:

```go
func (b *builder) paragraph(n ast.Node) {
    _, tight := n.(*ast.TextBlock)
    if img, link, ok := soleImage(n); ok {
        blk := Block{Kind: Image, Src: string(img.Destination), Alt: plainText(img, b.src), Link: link}
        blk.Video = link != "" && youtube.IsVideoLink(link)
        b.emit(n, blk, tight)
        return
    }
    b.emit(n, Block{Kind: Paragraph, Spans: b.inlines(n)}, tight)
}

func soleImage(n ast.Node) (*ast.Image, string, bool) {
    if n.ChildCount() != 1 {
        return nil, "", false
    }
    switch c := n.FirstChild().(type) {
    case *ast.Image:                          // ![Alt](src)
        return c, "", true
    case *ast.Link:                           // [![Alt](src)](link)
        img, ok := c.FirstChild().(*ast.Image)
        return img, string(c.Destination), ok && c.ChildCount() == 1
    }
    return nil, "", false
}
```

The result for the video above:

```go
Block{
    Kind:  Image,
    Src:   "https://img.youtube.com/vi/dQw4w9WgXcQ/maxresdefault.jpg",
    Alt:   "Video Title",
    Link:  "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
    Video: true,             // because the *link* is a YouTube video
    Line:  …, Align: …,      // as for every block (emit)
}
```

Note what decides `Video`: the **link**, not the image. Any picture
linked to a YouTube video gets the video treatment (66 % width, play
button); a YouTube still without a link is an ordinary image.

### 7.4 Finding the pixels: local files and the download cache

`render/images.go: ResolveImage` turns `Src` into a file path on disk,
for both the preview and the PDF:

```go
func ResolveImage(baseDir, src string) (string, bool) {
    if imgcache.Allowed(src) {            // a YouTube still:
        return imgcache.Path(src)         //   its downloaded copy, if there is one yet
    }
    if strings.Contains(src, "://") {     // any other remote image: not shown
        if !strings.HasPrefix(src, "file://") {
            return "", false
        }
        src = strings.TrimPrefix(src, "file://")
    }
    if p, err := url.PathUnescape(src); err == nil {
        src = p                           // "my%20image.png" → "my image.png"
    }
    if !filepath.IsAbs(src) {
        src = filepath.Join(baseDir, src) // relative to the document's folder
    }
    return src, true
}
```

**Only YouTube stills are downloaded.** `imgcache.Allowed` is
`youtube.IsThumbnail`: `https://` on `img.youtube.com` or `i.ytimg.com`
with a path starting `/vi/`. A Markdown file can otherwise make an editor
fetch any URL just by being opened — a tracking pixel tells the sender
who read the file, and when. Other remote images show their alt text.

`internal/imgcache` keeps the downloads in `<UserCacheDir>/mdedit/images`
(`~/.cache/mdedit/images` on Linux and FreeBSD,
`~/Library/Caches/mdedit/images` on macOS). Each file is named after a
hash of its URL plus the URL's extension (the PDF writer uses the
extension to tell the image type). It has two entry points:

| | Used by | Behaviour |
|---|---|---|
| `Fetch(src, done)` | the preview | Returns at once; downloads in a goroutine and calls `done` when the file is there |
| `Get(src)` | the PDF export | Downloads now and waits (the export needs the image before it can lay out the page) |

Both share the same rules (`imgcache.go: start`, `finish`, `download`,
`fetch`, `store`):

- **No duplicates:** a URL already being downloaded (`inflight`) isn't
  fetched again; the preview asks every frame until it arrives.
- **No hammering:** after a failure the URL isn't tried again for a
  minute (`retryAfter`). The preview keeps showing the placeholder or
  alt text meanwhile.
- **Fallback still:** `maxresdefault.jpg` doesn't exist for some older or
  low-resolution videos. On a 404, `download` fetches `hqdefault.jpg`,
  which exists for every video, and stores it under the original URL's
  name.
- **Checked and bounded:** at most 10 MB, a 15 s timeout, and the data
  must decode as an image (`image.DecodeConfig`) before it is kept.
- **Atomic:** the file is written as `….tmp` and renamed, so a reader
  never sees half an image.

### 7.5 Layout: size, alignment, placeholder, play button

`render/image.go: Ctx.image` turns the block into prims. The device tells
it the image's size in pixels through `Ctx.ImageSize`:

- **Preview** (`ui/preview.go: imageSize`): decodes the image through
  `Preview.image` (§7.6) — it needs the pixels anyway.
- **PDF** (`pdfout/pdf.go: imageSize`): `imgcache.Get` for a still, then
  `render.ImageFileSize`, which reads only the file header
  (`image.DecodeConfig`), cached by path and modification time. Formats
  the PDF can't embed report "no size".

```go
func (c *Ctx) image(b *mdmodel.Block, w float32, col color.NRGBA) (*Layout, float32) {
    pw, ph, ok := c.imageSize(b.Src)
    loading := !ok && b.Video && c.VideoControls
    if loading {
        pw, ph, ok = 16, 9, true             // keep the space of a 16:9 still
    }
    if !ok || pw == 0 || ph == 0 {
        return c.altText(b, w, col)           // "[image: Alt]", italic, clickable
    }
    iw, ih := c.fitImage(b, float32(pw), float32(ph), w)
    x := alignedX(b.Align, w, iw)             // <div align="center"> etc.
    l := &Layout{Height: ih}
    if loading {
        l.Prims = append(l.Prims, Prim{Kind: PRect, X: x, W: iw, H: ih, Radius: 0.4 * c.Em, Color: videoPlaceholder})
    } else {
        l.Prims = append(l.Prims, Prim{Kind: PImage, X: x, W: iw, H: ih, URL: b.Src})
    }
    if b.Video && c.VideoControls {
        l.Prims = append(l.Prims, playButton(x, iw, ih)...)
    }
    if b.Link != "" {
        l.Prims = append(l.Prims, Prim{Kind: PLink, X: x, W: iw, H: ih, URL: b.Link})
    }
    return l, min(ih, c.Em)                   // "baseline", for a list bullet beside the image
}
```

`VideoControls` is set by the preview only: the PDF gets neither the
placeholder (it waits for the download instead) nor the play button (it
can't play anything; the still is a link to the video).

**Size** (`fitImage`):

```go
iw, ih := pw*c.Em/16, ph*c.Em/16               // image pixels count as CSS pixels: 16 per em
if b.Video {
    iw, ih = w*videoShare, w*videoShare*ph/pw  // videos: 66 % of the column, keep the aspect ratio
}
if iw > w {
    iw, ih = w, ih*w/iw                         // never wider than the column
}
if c.MaxImageHeight > 0 && ih > c.MaxImageHeight {
    iw, ih = iw*c.MaxImageHeight/ih, c.MaxImageHeight  // PDF: never taller than a page
}
```

Counting image pixels as CSS pixels makes an image the same size
*relative to the text* everywhere: a 400 × 300 PNG is 25 em wide, which
is 400 device pixels on a normal screen (em = 16 px), 800 on a Retina
screen (em = 32 px), and 262.5 pt in the PDF (em = 10.5 pt).

**A worked video**, in a preview column 800 px wide at em = 16 px, with
a 1280 × 720 still:

```
fitImage:     iw = 800 × 0.66 = 528,   ih = 528 × 720 / 1280 = 297
alignedX:     x = 0 (left), 136 (centre), 272 (right)
playButton:   bw = max(528 × 0.14, 24) = 73.9   bh = 0.7 × bw = 51.7
              centred at (x + 264, 148.5), corner radius 0.28 × bh = 14.5
              triangle half-height t = 0.22 × bh = 11.4

Layout{Height: 297, Prims: [
    PImage  x 0      y 0      528 × 297   URL: …/maxresdefault.jpg
    PRect   x 227    y 122.6  73.9 × 51.7 red, rounded
    PPoly   (254.9, 137.1) (276.5, 148.5) (254.9, 159.9)   white triangle
    PLink   x 0      y 0      528 × 297   URL: https://www.youtube.com/watch?v=…
]}
```

The play button is not an icon file: it is a red rounded rectangle and a
white triangle, drawn like any other shape (§6.8), so it is sharp at
every size.

### 7.6 The preview: decode, cache, arrive late, click

`ui/images.go: Preview.image` is called both by layout (for the size)
and by `paintImage` (for the pixels):

```go
func (p *Preview) image(src string) *imageEntry {
    path, ok := render.ResolveImage(p.BaseDir, src)
    if !ok {
        if imgcache.Allowed(src) {
            imgcache.Fetch(src, p.imageArrived)     // not downloaded yet: start (once)
        }
        return nil
    }
    st, err := os.Stat(path)
    if err != nil || st.IsDir() {
        return nil
    }
    e := p.cachedImage(path, st.ModTime().UnixNano()) // decoded again only if the file changed
    if e.failed {
        return nil
    }
    return e
}
```

Each `imageEntry` holds the `paint.ImageOp` (the pixels) and the size. A
file that fails to decode is remembered as `failed`, so a broken image
costs one attempt, not one per frame. Because entries are keyed by
modification time, re-saving an image in another program shows up in
the preview without restarting.

**A video arriving late**, frame by frame:

1. **Open the document.** Layout asks for the still's size: `ResolveImage`
   finds no cached file, `Fetch` starts a goroutine, `image` returns nil.
   `Ctx.image` sees a video without a size → `loading` → a dark 16:9
   placeholder with the play button. The layout is cached like any other.
2. **Meanwhile** the user types; the placeholder stays; no frame waits
   for the network.
3. **The download finishes** (on the goroutine): `imageArrived` sets
   `imagesChanged` (an atomic, because it's another goroutine) and calls
   `Invalidate`, which asks Gio for a frame.
4. **Next frame:** `Preview.measure` sees `imagesChanged.Swap(false)` and
   drops the layout cache. The block is laid out again; this time
   `ResolveImage` finds the file, `decodeImage` decodes it, the size is
   real, and the layout has a `PImage` instead of the placeholder.
   `paintImage` draws it (§6.10).

An ordinary remote still without a video link has no placeholder: it
shows its alt text until the download arrives, then the image.

**Clicking** the still: `ui/preview.go: block` registers a click area
over each block; `handleClicks` finds the `PLink` under the pointer
(`linkAt`) and calls `OnLink`, which is `ui/files.go: followLink`:

```go
func (a *App) followLink(url string) {
    switch {
    case strings.HasPrefix(url, "#"):
        a.jumpToHeading(url)        // a heading in this document
    case strings.Contains(url, ":"):
        a.openExternal(url)         // https://www.youtube.com/watch?v=… → the browser
    default:
        a.openRelative(url)         // a file next to this one: Markdown opens in mdedit,
                                    // anything else in its default application
    }
}
```

`openExternal` → `ui/widgets.go: openURL` runs `open` on macOS,
`xdg-open` on Linux and the BSDs, and `rundll32 url.dll,…` on Windows.
The hand cursor over the still comes from the same `PLink`
(`ui/paint.go: linkCursor`).

### 7.7 The PDF

The PDF uses the same `Block` and the same `Ctx.image`, with
`VideoControls` off, so a video is just its still, linked to the watch
page.

- **Before layout:** `pdfout/pdf.go: imageSize` calls `imgcache.Get`, so
  the export **waits** for any still not yet downloaded. Offline, the
  download fails and the block falls back to its alt text — a clickable
  `[image: Video Title]` linking to the video.
- **Page breaks:** an image is never split; `Ctx.image` sets no
  `Layout.Breaks`, so an image that doesn't fit moves to the next page
  (`pdfout/paging.go: place`), and `MaxImageHeight` (the page height
  minus margins) guarantees it fits on an empty one.
- **Drawing** (`pdfout/draw.go: drawImage`): the image file is embedded
  as it is (`fpdf.ImageOptions` with the type from the extension, PNG,
  JPEG or GIF) and placed at the prim's box; the viewer scales it. fpdf
  embeds each file once, however often it appears.
- **The link** (`pdfout/draw.go: drawLink`): the `PLink` prim becomes a
  PDF link annotation over the image (`LinkString`), so clicking the
  still in a PDF viewer opens the video too. A `#heading` link becomes an
  internal link to that heading's page instead.

### 7.8 When an image doesn't show

| What you see | Why | Where |
|---|---|---|
| `[image: Alt]` for a remote image | Only YouTube stills are downloaded | `render/images.go: ResolveImage`, `youtube.IsThumbnail` |
| `[image: Alt]` for a local file | Path wrong (relative to the `.md` file?), or not PNG/JPEG/GIF | `ResolveImage`, `ui/images.go: decodeImage` |
| `[image: Alt]` in the middle of a line | Pictures are drawn only when alone in their paragraph | `mdmodel/parse.go: soleImage` |
| The dark video placeholder stays | Download failed; retried after a minute | `internal/imgcache` (delete the cache folder to start over) |
| A video drawn as an ordinary image | The *link* isn't a YouTube video URL | `youtube.IsVideoLink` |
| The PDF shows alt text for a video | No network during the export | `pdfout/pdf.go: imageSize` |

---

## 8. The browser version (WebAssembly)

mdedit also runs in a web browser. It is **the same program**: `make wasm`
compiles `cmd/mdedit` with `GOOS=js GOARCH=wasm`, and Gio has a browser
backend that draws the window into a `<canvas>` with WebGL — through the
same shaders as on the desktop (§6.8). Parsing, layout, maths,
highlighting, the preview, the PDF writer and the UI are untouched.

What a web page *can't* do is touch the computer: Go's `os.ReadFile` and
friends return "not implemented" in a browser, and `exec.Command` can't
run anything. So everything that reaches outside the program sits behind
two seams, each with a desktop and a browser side chosen by **build tags**:

| Seam | Desktop (`//go:build !js`) | Browser (`//go:build js`) |
|---|---|---|
| `internal/ui/platform_*.go` | `platform_desktop.go`: mdedit's file dialog, `os` files, `settings.json`, crash-recovery file, `open`/`xdg-open` | `platform_js.go`: the browser's file picker, downloads, `localStorage`, `window.open` |
| `internal/vfs` | `vfs_disk.go`: the disk | `vfs_mem.go`: a map in memory |

The two files of a pair define the **same functions and types**; the
rest of the code calls them without knowing which one it gets. The
desktop functions were moved into `platform_desktop.go` unchanged, and
the desktop screenshots and PDFs are pixel-identical to before the browser
version existed.

### 8.1 The platform seam in `internal/ui`

What each side provides:

| Function / type | Desktop | Browser |
|---|---|---|
| `loadInitial(path)` | Read the file named on the command line | Restore the text from `localStorage` |
| `showOpenDialog()` | `NewFileDialog` (mdedit's own dialog) | A hidden `<input type="file">`, clicked |
| `save(then)`, `saveAs(then)` | Write the file (asking for a name first if needed) | Download `name.md` |
| `exportDialog()` | File dialog, then `writePDF` | Make the PDF on a goroutine, then download it |
| `openURL(url)` | `open` / `xdg-open` / `rundll32` | `window.open(url, "_blank")` |
| `readSettings`, `writeSettings` | `<UserConfigDir>/mdedit/settings.json` | `localStorage["mdedit.settings"]` |
| `saveRecovery()` | Unsaved text to the cache folder on close | Nothing: `autosave` has kept it |
| `autosave()` | Nothing | Text, saved text and name to `localStorage`, on every change |
| `quitCommands()` | File → Quit | None: a tab is closed by the browser |
| `platformFonts()` | None: the system's fonts cover ⌘ and arrows | STIX Two Math (`tex.MathFont`), for ⌘ ⇧ ← … |
| `platformState`, `handleArrivals()` | Empty | Results from browser callbacks (below) |

Three of these are called from shared code: `New` builds the shaper with
`append(gofont.Collection(), platformFonts()...)`, `reparse` ends with
`a.autosave()`, and `handleOpenRequest` (every frame) starts with
`a.handleArrivals()`. `fileMenu` appends `a.quitCommands()`.

**Fonts.** On the desktop, Gio's shaper also loads the system's fonts, so
a ⌘ in the status bar is found in some system font. A web page has no
system fonts; without help, ⌘ is the missing-glyph box (§6.2). The browser
side adds the maths font, which has ⌘, ⇧ and the arrows.

### 8.2 Asynchronous results: picker, file reading, PDF export

On the desktop, opening a file is a function call. In the browser it is a
chain of events that arrive later, from JavaScript:

```
showOpenDialog ─▶ <input type=file>.click()        (needs a recent click or key press,
                                                    or the browser ignores it)
      … the user picks a file …
fileChosen (JS "change" event) ─▶ file.text()      (a Promise)
      … the browser reads it …
textRead (Promise resolved) ─▶ platform.opened = {name, text}; invalidate()
      … next frame …
handleArrivals ─▶ a.load(name, text)
```

JavaScript callbacks are Go methods wrapped once with `js.FuncOf`
(`initCallbacks`: `js.FuncOf(a.fileChosen)` — a method value, no
closure). They run outside the frame loop, so they don't touch the editor:
they store the result in `a.platform` under its mutex and ask for a frame;
`handleArrivals` applies it on the frame goroutine. This is the same
pattern as `ui/app.go: RequestOpen` for files the macOS Finder asks to open.

**Export PDF** works the same way: `exportDialog` starts
`exportInBackground` on a goroutine (the export may wait for video stills
to download, §7.7, and blocking the frame goroutine on the network would
freeze the page), which writes the PDF into a buffer, downloads it and
reports through `deliver(status)`.

**Downloads** (`download`) wrap the bytes in a `Blob`, give it a temporary
URL, and click a hidden `<a download="name">` pointing at it. The URL is
released a minute later (`revokeURL`).

### 8.3 Storage

- **The document** is written to `localStorage` on every change
  (`autosave`): `mdedit.text` (the editor's text), `mdedit.saved` (as last
  opened or downloaded, so the "unsaved" dot survives a reload) and
  `mdedit.name`. `loadInitial` reads all three **before** calling `load`,
  because `load` itself autosaves and would overwrite them.
- **Settings** go to `mdedit.settings`, with the same JSON as
  `settings.json`; the first visit follows the browser's dark mode
  (`gioui.org/x/pref/theme` asks the `prefers-color-scheme` media query).
- **Images** (`internal/vfs`): `imgcache` stores downloaded stills with
  `vfs.WriteFile`; `render.ImageFileSize`, `ui/images.go` and
  `pdfout.registerImage` read them with `vfs.ModTime` and `vfs.Open`. On
  the desktop that is the disk; in the browser, a map that lasts as long
  as the page. Local images next to a document resolve to paths that
  aren't in the map, so they show their alt text (§7.8).
- **Network:** `net/http` works in the browser (Go uses `fetch`). The
  browser only lets a page read images from another site if that site
  allows it; YouTube's still servers send `Access-Control-Allow-Origin: *`.

The PDF writer needed one change for this: fpdf used to open image files
itself (`ImageOptions(path)`), which can't work on a map in memory.
`pdfout/draw.go: registerImage` now reads the file through `vfs` and
hands fpdf a reader (`RegisterImageOptionsReader`); `ImageOptions` then
finds it by name.

### 8.4 The web page and keys

`web/index.html` loads Go's `wasm_exec.js`, fetches `mdedit.wasm` and
starts it; Gio puts its canvas into `<div id="giowindow">`.

Gio passes key presses to mdedit but doesn't stop the browser from
acting on them too: Ctrl+S would also open "Save page", Ctrl+F the
browser's search. The page listens to `keydown` first and cancels the
browser's default for Ctrl/⌘ combinations, except copy, cut, paste and
zoom (and paste-as-text and two developer-tools keys with Shift).
Cancelling the default doesn't stop the event, so Gio still gets it.
Some keys can't be taken from the browser at all (Ctrl/⌘+N, T, W).

Gio's browser backend picks the shortcut modifier from the browser's
platform: ⌘ on a Mac, Ctrl elsewhere (`key.ModShortcut` is a variable in
the `js` build), so the menus and status bar show the right one.

**Clipboard:** Gio uses the browser's Clipboard API, which browsers only
offer on HTTPS pages and on `localhost`. Over plain HTTP from another
machine, mdedit's copy and paste do nothing — serve it over HTTPS.

### 8.5 Building, serving, installing

`make wasm` (Makefile):

1. `GOOS=js GOARCH=wasm go build ./cmd/mdedit` → `bin/web/mdedit.wasm`
   (about 30 MB: the Go runtime, Gio, goldmark, chroma's lexers and the
   fonts);
2. copies Go's `wasm_exec.js` (from `$(go env GOROOT)/lib/wasm`, or
   `misc/wasm` for older Go), `web/index.html` and the icon;
3. compresses: `.gz` always, `.br` if `brotli` is installed (about 8 and
   5 MB);
4. builds `bin/mdedit-web` for this machine, without cgo.

`cmd/mdedit-web` is a small static server: for each request it sends the
best compressed copy the browser accepts (`files.pick`), with the
*original* file's content type (`application/wasm`, not `application/gzip`)
and `Content-Encoding`; `Cache-Control: no-cache` makes browsers check for
a new build on every visit (an unchanged file costs a `304`). With
`-tls-cert` and `-tls-key` it serves HTTPS.

`make install-jail` (FreeBSD host, as root) copies `bin/web` into the
jail's web folder, `mdedit-web` and `assets/freebsd/mdedit_web` (the rc.d
script) into the jail, enables the service with `sysrc -R <jail root>`,
and restarts it with `jexec` if `jls` shows the jail running. The rc.d
script runs the server under `daemon(8)` as user `www`, with its pid file
in `/var/run/mdedit_web/` and its log in `/var/log/mdedit_web.log`, both
created for that user before start (`mdedit_web_prestart`).

### 8.6 Adding something that touches the outside world

If a new feature reads or writes files, runs programs, or opens
anything: put the platform-specific part in **both** `platform_desktop.go`
and `platform_js.go` (or use `vfs` for data files), keep the shared code
calling it, and run `make test` — the `GOOS=js` vet step fails if the
browser side is missing. Then try it with `make wasm && make serve`.

---

## 9. The layers in detail

### 9.1 `internal/ui` — the window

| File | What's in it |
|---|---|
| `app.go` | `App` (all window state), `New`, `load`, `reparse`, `Run`, crash-recovery save, window title |
| `window.go` | The per-frame `layout`; input order; right-click context menu; overlay order |
| `panes.go` | Editor / preview / split view, the draggable divider, `editorView`, `followCaret` |
| `commands.go` | `command` type, **all menus** (`fileMenu` … `helpMenu`), the toolbar list, `cmd*` actions |
| `edits.go` | `applyEdit` / `setEditorState` (one undo step), parametrised edits (`headingLevel`, `inlineMarker`, `tableSize`, `videoLink` …), cut/copy |
| `keys.go` | Keyboard shortcuts, Enter/Tab interception, macOS ⌘ keys, wrapped-line navigation |
| `files.go` | New/Open/Save/Save As/Export PDF, the "unsaved changes" flow (`flow`, `action`), link following |
| `menubar.go`, `popup.go` | Menu bar titles; drop-down and context menu items |
| `toolbar.go`, `statusbar.go` | Icon toolbar; status line |
| `findbar.go`, `findbar_view.go` | Search/replace logic and view, match highlighting |
| `filedialog.go`, `confirm.go`, `tabledialog.go`, `videodialog.go` | Modal dialogs (all implement `dialog`) |
| `preview.go`, `paint.go`, `textpaint.go`, `images.go` | The preview: list, layout cache, painters per `Prim` kind, text shaping, image cache |
| `widgets.go` | Shared building blocks: `button`, `framed`, `vspace`/`hspace`, `modal`, `keyPressed` |
| `platform_desktop.go`, `platform_js.go` | Everything that differs between desktop and browser: Open/Save/Export, settings storage, links, Quit, autosave, fonts (§8) |
| `colors.go`, `theme.go` | The window's colours (`uiColors`: `lightColors`, `darkColors`, the current one in `col`); switching theme (`setDark`); `settings.json` |

**Gio style used throughout:** no closures. A piece of UI is a named method
with the `layout.Widget` signature, and repeated rows are small types with a
`layout` method:

```go
// ui/toolbar.go
type toolbarButton struct {
    a *App
    c *command
}

func (b toolbarButton) layout(gtx layout.Context) layout.Dimensions {
    if b.c.click.Hovered() {
        b.a.hint = b.c.tooltip()
    }
    return layout.Inset{Right: 2}.Layout(gtx, iconButton{click: &b.c.click, icon: b.c.icon}.layout)
}
```

**Multi-step file operations** ("save changes first?" → save dialog →
"overwrite?" → write) are a small state machine instead of nested
callbacks: `files.go` records the pending `action` in `App.flow`, and the
dialogs report back to named methods such as `discardAnswered`,
`saveAsChosen`, `overwriteAnswered`.

**Themes** have three parts, and `theme.go: setDark` switches all three at
once:

1. `col` (`colors.go`) — the window chrome: toolbar, menus, dialogs, find
   bar, match highlights. UI code always writes `col.toolbarBg`, never a
   literal colour.
2. `a.th.Palette` — Gio's `material` palette, which colours text, editors,
   check boxes and scroll bars.
3. `Preview.SetTheme(&render.Light / &render.Dark)` — the preview's
   `render.Theme` (text, links, borders, code backgrounds and the chroma
   style for code); it clears the layout cache so every block is laid out
   again in the new colours.

At start-up `startDark` uses the saved choice in
`<UserConfigDir>/mdedit/settings.json`, or else asks the system
(`gioui.org/x/pref/theme`, which only knows on macOS and Windows).
`cmdToggleTheme` (View → Dark Theme, Ctrl+Shift+D) flips it and saves it.
The PDF exporter always uses `render.Light`.

### 9.2 `internal/edit` — editing commands as pure functions

| File | Commands |
|---|---|
| `edit.go` | `State`, inline markers (`ToggleInline`), `Newline` (list continuation), insertions (`Table`, `CodeBlock`, `MathBlock`, `Link`, `Rule`, `Video`), `ExpandTabs` |
| `block.go` | Line-based commands via the `lineMapper` interface: `ToggleList`, `ToggleQuote`, `SetHeading`, `Tab`, `Outdent` |
| `align.go` | `SetAlign` (the `<div align>` wrapper), using `lineDoc` |
| `toc.go` | `BuildTOC`, `InsertTOC`, `UpdateTOC`, `FindTOC` |
| `search.go` | `Query`, `FindAll`, `NextMatch`, `PrevMatch`, `ReplaceSelection`, `ReplaceAll` |

Line-based commands implement one method:

```go
// edit/block.go
type lineMapper interface {
    mapLine(line string) string
}

type headingMapper struct {
    level  int
    single bool
}

func (m headingMapper) mapLine(l string) string { … }

func SetHeading(s State, level int) State {
    …
    return s.mapLines(headingMapper{level: level, single: !s.multiLine()})
}
```

### 9.3 `internal/mdmodel` — Markdown to blocks

| File | What's in it |
|---|---|
| `model.go` | `Document`, `Block`, `Span`, `Decor`, `TableData`, kinds and alignments |
| `parse.go` | `Parse`; the `builder` that walks goldmark's tree (`block` dispatcher, one method per node kind); spacing (`computeGaps`) |
| `inline.go` | Inline spans: `inlineRun.node` dispatcher, raw inline HTML (`<u>`, `<br>`), task boxes |
| `html.go` | HTML blocks, the `<div align="…">` wrapper |
| `math.go` | goldmark extension for `$$…$$` blocks and `$…$` inline maths |
| `highlight.go` | Syntax highlighting with chroma (token types only, no colours), language aliases |

The block dispatcher is the place new block types start:

```go
// mdmodel/parse.go
func (b *builder) block(n ast.Node) {
    switch n := n.(type) {
    case *ast.Blockquote:
        b.quote(n)
    case *ast.List:
        b.list(n)
    case *ast.Heading:
        b.heading(n)
    case *ast.Paragraph, *ast.TextBlock:
        b.paragraph(n)
    case *mathBlock:
        b.emit(n, mathBlockOf(b.lines(n.Lines())), false)
    …
    default:
        b.children(n)
    }
}
```

`emit` stamps every block with its source line, its containers (`Decor`)
and any active alignment.

### 9.4 `internal/render` — blocks to primitives

| File | What's in it |
|---|---|
| `render.go` | `Ctx`, `Prim`, `Layout`, `Theme` (`Light`, `Dark`), list markers, paragraph/heading/code layouts, maths block, `mathPrims` |
| `block.go` | `Ctx.Block` — containers (`frame`), the `content` dispatcher, offsets, quote bars |
| `inline.go` | Inline layout: `lineBreaker` (word wrapping, inline maths boxes) and `lineEmitter` (runs → prims: text, code background, underline, strike, links) |
| `table.go` | `tableLayout`: column widths, rows, grid lines |
| `image.go` | Images and YouTube videos (66% width, play button, placeholder) |
| `images.go` | `ResolveImage` (local paths, cached YouTube stills), `ImageFileSize` |

The content dispatcher is the second place new block types touch:

```go
// render/block.go
func (c *Ctx) content(b *mdmodel.Block, w float32, col color.NRGBA) (*Layout, float32) {
    switch b.Kind {
    case mdmodel.Heading:
        return c.heading(b, w, col)
    case mdmodel.Code, mdmodel.HTML:
        return c.code(b, w)
    …
    }
    return c.paragraph(b.Spans, w, col, b.Align)
}
```

### 9.5 `internal/pdfout` — primitives to PDF

| File | What's in it |
|---|---|
| `pdf.go` | `Write`, `exporter`, `placeBlock` (headings kept with the next lines), link targets |
| `paging.go` | `place` / `fillPage`: splitting a block across pages at `Layout.Breaks`; `segment` |
| `draw.go` | `drawPrim` dispatcher and one function per `Prim` kind |
| `fonts.go` | The embedded Go fonts, measuring, and **fallback glyphs** (⌘, ⇧, ✓ … drawn as outlines from the maths font) |

### 9.6 `internal/tex` — maths

| File | What's in it |
|---|---|
| `parse.go` | TeX parser → node tree (`nSym`, `nFrac`, `nScripts` …), `rows` for `\\` and `&` |
| `commands.go` | **The command table** (`handlers`, filled in `init`) and one parser method per command |
| `symbols.go` | Symbol tables (Greek, operators, relations, arrows), big operators, functions, maths alphabets (`\mathbb` …) |
| `layout.go` | `Layout` (cached), `Box`/`Item`, atom spacing (TeX's table), glyph boxes, `layoutNode` dispatcher |
| `scripts.go`, `fraction.go`, `delims.go`, `accents.go`, `array.go` | Sub/superscripts and limits; fractions and roots; stretchy delimiters; accents; matrices/cases/aligned |
| `font.go` | The embedded STIX Two Math font (sfnt), glyph metrics and outlines, `Fallback` |

`tex.Layout(src, display)` returns a `Box` of `Item`s in em units;
`render/render.go: mathPrims` turns them into `PGlyph`/`PRect`/`PPoly` prims.

### 9.7 `internal/imgcache` and `internal/youtube`

`imgcache.Fetch(src, done)` downloads in the background (the preview calls
it from `ui/images.go: Preview.image` and redraws in `imageArrived`);
`imgcache.Get(src)` downloads synchronously (the PDF calls it from
`pdfout/pdf.go: imageSize`). Only YouTube stills are allowed
(`youtube.IsThumbnail`). Failures aren't retried for a minute. §7 follows
images and videos through all layers.

---

## 10. Recipes: where to add things

### Add an editing command (menu item, shortcut, maybe a toolbar icon)

1. Write the text transformation in `internal/edit` as
   `func(State) State` (or a type with an `apply` method if it needs
   parameters) and **test it** in `edit/edit_test.go` with the
   `"text [selection]"` notation.
2. In `ui/commands.go`, add it to a menu with `a.editCommand(label, key,
   mods, f)`; for parameters, add a small type with an `apply` method in
   `ui/edits.go` (like `headingLevel`).
3. A shortcut is just the `key`/`mods` arguments; `handleShortcuts` picks it
   up automatically. Check it doesn't clash with an existing one (search
   `commands.go` for the key).
4. For a toolbar icon: keep the command in `commands` (the struct), set
   `.icon = mustIcon(icons.…)` in `buildToolbar` and add it to
   `a.toolbarCmds`. Icons come from
   `golang.org/x/exp/shiny/materialdesign/icons`. Menu items show the icon
   automatically.
5. Add a UI test in `ui/*_test.go` using the harness
   (`h.press("B", key.ModShortcut)`, `h.expect(…)`).

### Add a dialog

Copy `ui/tabledialog.go` or `ui/videodialog.go`: a type with
`Layout(gtx, th)`, a `Done` callback field set to a **named method** of
`App`, input handled first (`handleInput`), then `modal(gtx, d.card)`. Open
it by assigning `a.dialog = NewXDialog(a.xChosen)`. If the result edits the
text, set `a.pendingEdit` in the callback (dialogs have no `gtx`).

### Add a Markdown block type

Touch these, in order — the compiler won't remind you about the switches:

1. `mdmodel/model.go`: a new `Kind` (and fields on `Block` if needed).
2. `mdmodel/parse.go: block`: a case that `emit`s it (if goldmark doesn't
   parse it, add a parser like `mdmodel/math.go`).
3. `render/block.go: content`: a case that lays it out, returning a
   `*Layout` and the baseline of its first line. Add `Breaks` if the PDF
   may split it across pages.
4. If you need a new kind of shape, see the next recipe.
5. Test: `mdmodel/*_test.go` for parsing, a render or PDF test, and a
   screenshot scene (§11).

### Add an inline style or inline element

`mdmodel/model.go: Span` (a field) → `mdmodel/inline.go: inlineRun.node` (a
case) → `render/inline.go: lineEmitter.text` (draw it) — and remember
`Span.sameStyle` in `model.go`, or adjacent spans will be merged.

### Add a new primitive kind

A new `Prim.Kind` must be handled in **three** switches:

1. `render/render.go`: the `PrimKind` constants (and `Prim.Top` if its
   extent isn't `Y`).
2. `ui/paint.go: paintPrim` — screen.
3. `pdfout/draw.go: drawPrim` — PDF (decide which page segment it belongs
   to: `seg.hasBaseline` or `seg.hasTop`).

Also `render/block.go: offsetPrim` if it has coordinates other than X/Y
(like `PPoly.Points`).

### Add a TeX command

- A symbol: add it to `tex/symbols.go: symbols` (with its class: ord, bin,
  rel …).
- A function name like `\sin`: `functions` (limits or not).
- A command with arguments: a parser method in `tex/commands.go` registered
  in `init`'s `handlers` map, a node type in `tex/parse.go` if needed, and a
  case in `tex/layout.go: layoutNode`.
- Test: add the formula to `tex/tex_test.go: corpus`.

### Add a keyboard behaviour inside the editor

If Gio's editor already handles the key, you must take it **before** the
editor does: add a handler called from `ui/window.go: handleInput` that asks
for the key with `key.Filter{Focus: &a.editor, Name: …}` (see
`ui/keys.go: handleTab`).

---

## 11. Finding bugs

### Symptom → where to look

| Symptom | Start here |
|---|---|
| Wrong Markdown structure (a list not detected, wrong nesting, missing block) | `mdmodel/parse.go` (`block`, `list`, `emit`); dump blocks with `go test ./internal/mdmodel -run TestParseTestFile -v` |
| Wrong inline styling (bold/links/code/maths spans) | `mdmodel/inline.go: inlineRun.node` |
| Spacing between blocks | `mdmodel/parse.go: gapBetween` |
| Indentation, quote bars, list bullets | `render/block.go: frame`, `render/render.go: marker` |
| Text wrapping, line height, inline code box | `render/inline.go: lineBreaker`, `lineEmitter` |
| Looks right in the preview but not in the PDF (or vice versa) | The painters: `ui/paint.go` vs `pdfout/draw.go`; measuring: `ui/textpaint.go: measure` vs `pdfout/fonts.go: measure` |
| Something cut or duplicated at a PDF page break | `pdfout/paging.go` (`place`, `lastBreak`, `segment`) and the block's `Layout.Breaks` |
| Boxes instead of characters in the PDF | `pdfout/fonts.go: needsFallback` — the character is in neither the Go font nor STIX |
| Maths: wrong position/size | `tex/scripts.go`, `fraction.go`, `delims.go`; the constants at the top of `tex/layout.go` |
| Maths: wrong symbol or "red" command | `tex/symbols.go`, `tex/commands.go: init` |
| A key does nothing, or does the wrong thing | Who takes it first: `ui/window.go: handleInput` order, `ui/keys.go`; is a dialog or the find bar focused? Is the shortcut in `commands.go`? |
| A click does nothing | Was the clickable laid out last frame? Is an overlay (dialog, popup dismiss area, `blockPointer`) on top of it? |
| A colour wrong in one theme only | Window chrome: `ui/colors.go`; Gio widgets: the palettes in `ui/theme.go`; preview: `render.Light` / `render.Dark`; code tokens: the chroma style named in `Theme.CodeStyle` |
| Preview doesn't update | `ui/app.go: reparse` → `Preview.SetDocument`; the cache reset in `ui/preview.go: measure` |
| Preview doesn't scroll with the caret | `ui/panes.go: followCaret`, `Block.Line` from `mdmodel/parse.go: emit` |
| Images / YouTube stills missing | `render/images.go: ResolveImage`, `ui/images.go`, `internal/imgcache` (cache in `~/.cache/mdedit/images`, `~/Library/Caches/mdedit/images` on macOS) |
| Crash with a stack trace | The trace names the function; check slices that change while being iterated (the file-dialog crash was that) and nil cache entries (the image crash was that) |

### Tools

- **Run everything:** `make test`. This includes the style check
  (`internal/lint`), so a style violation fails the build.
- **Export a PDF without the UI:** `./bin/mdedit -pdf file.md -o out.pdf`.
- **Screenshots without a display:** `MDEDIT_SHOT=/tmp/shots go test
  ./internal/ui -run TestScreenshot` renders the real window offscreen
  (split view, menus, dialogs, find bar, maths, videos …, and the same in
  the dark theme as `dark-*.png`) to PNG files.
  Add a scene in `ui/screenshot_test.go` when you add something visual.
- **The browser version:** `make wasm && make serve`, then open
  `http://localhost:8080`. The browser's developer tools (F12) show Go's
  panics and `println` output in the console.
- **The UI test harness** (`ui/harness_test.go`) drives the real `App`
  through Gio's input router: `h.press(key, mods)`, `h.rightClick(x, y)`,
  `h.run(cmd)`, `h.setText(text, start, end)`, `h.expect(text)`,
  `h.expectSelection(caret, anchor)`. Most bugs can be reproduced there
  first, then fixed.
- **Regression check for rendering changes:** export the test documents in
  `internal/mdmodel/testdata` with the old and the new binary, rasterise
  the pages and compare pixels. PDF bytes aren't comparable (fpdf orders
  font subsets differently on each run), pixels are.

### Gotchas

- **Coordinates:** y points down everywhere (Gio, `render`, `tex`); text and
  maths glyph `Y` is the **baseline**, rectangles use their top-left corner.
- **Units:** `render` works in device units given by `Ctx.Em` (pixels on
  screen, points in PDF); `tex` works in em; image pixels count as CSS
  pixels (16 per em).
- **Rune offsets:** `edit.State` and Gio's editor use rune offsets, not
  bytes. Mixing them up breaks on any non-ASCII text.
- **Tabs:** Gio can't display tab characters, so the editor converts them
  to spaces (`edit.ExpandTabs`) on load and paste.
- **Caches:** the preview caches layouts per block and resets them on a new
  document, a width or font-size change, or an image arriving
  (`ui/preview.go: measure`). `tex.Layout` and `mdmodel.Highlight` cache by
  source text.
- **Focus:** Ctrl/⌘+Z undoes in whichever editor has the focus — the main
  editor or a field of the find bar or a dialog.
- **Theme in tests:** `ui/main_test.go: TestMain` points `configDir` at a
  temporary directory with the light theme saved, so tests don't depend on
  the machine's dark mode or touch your real settings.
- **Keys after a menu in tests:** Gio delivers keys only to filters
  registered in the *previous* frame, and shortcuts aren't polled while a
  menu is open. In the harness, call `h.frame()` after closing a menu
  before pressing a shortcut.
- **Two builds:** `platform_js.go` and `vfs_mem.go` only compile for the
  browser (`//go:build js`), their desktop twins only for the desktop
  (`!js`). A function added to one must be added to the other.
  `make test` vets the browser build too (`GOOS=js GOARCH=wasm go vet`),
  which catches a missing twin; the UI tests themselves only run on the
  desktop.
- **macOS builds** need `CC=clang` (the Makefile sets it); a Homebrew GCC
  can't compile Gio's Objective-C.

---

## 12. Code style

Enforced by `internal/lint/style_test.go`, which runs with `make test`:

- **No anonymous functions.** Use named functions, methods, and method
  values (`a.followLink`, `headingLevel(2).apply`). For Gio, a widget is a
  method with the signature `func(layout.Context) layout.Dimensions`.
- **At most 4 levels of indentation** on any line.
- **At most 40 lines per function.** Split by responsibility; small types
  with methods replace closures that captured state.

Other conventions: keep layers pure where they can be (`edit`, `mdmodel`,
`render`, `tex` don't touch the UI), write a test before fixing a bug, and
keep the preview and the PDF on the shared `render` path.
