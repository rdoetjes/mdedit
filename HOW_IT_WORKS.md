# How mdedit works

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
| `cmd/mdedit` | `main`: flags, `-pdf` export, opens the window | ui, mdmodel, pdfout |
| `internal/ui` | Everything you see and click: window, editor, preview painter, menus, toolbar, dialogs, find bar, keys | edit, mdmodel, render, pdfout, tex, imgcache, youtube |
| `internal/edit` | Formatting commands as **pure functions** on `State` (text + selection): bold, lists, headings, align, tables, TOC, search/replace, Enter/Tab | mdmodel (for TOC), youtube |
| `internal/mdmodel` | Parses Markdown into a flat list of styled `Block`s; maths syntax; syntax highlighting | goldmark, chroma, youtube |
| `internal/render` | Lays out a `Block` at a given width into a `Layout` of positioned `Prim`s | mdmodel, tex, imgcache |
| `internal/tex` | TeX maths typesetter; embeds the STIX Two Math font; glyph outlines | x/image/font/sfnt |
| `internal/pdfout` | Paints `Layout`s onto PDF pages, splitting blocks at page breaks | render, tex, imgcache, fpdf |
| `internal/imgcache` | Downloads YouTube stills into an on-disk cache (async for the preview, sync for PDF) | youtube |
| `internal/youtube` | YouTube URLs and video-ID parsing | — |
| `internal/lint` | A test enforcing the code style (see §9) | — |
| `tools/mkicon` | Draws the application icon PNGs | x/image/vector |

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
    Color color.NRGBA   // syntax highlighting
}
```

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
   glyph outlines). Back in `Run`, `e.Frame` sends it to the GPU.

The PDF export takes the same path from step 7 on, with
`pdfout/pdf.go: Write` → `placeBlock` → `Ctx.Block` →
`pdfout/paging.go: place` (page splitting at `Layout.Breaks`) →
`pdfout/draw.go: drawPrim`.

---

## 6. The layers in detail

### 6.1 `internal/ui` — the window

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
| `widgets.go` | Shared building blocks: `button`, `framed`, `vspace`/`hspace`, `modal`, `keyPressed`, colours |

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

### 6.2 `internal/edit` — editing commands as pure functions

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

### 6.3 `internal/mdmodel` — Markdown to blocks

| File | What's in it |
|---|---|
| `model.go` | `Document`, `Block`, `Span`, `Decor`, `TableData`, kinds and alignments |
| `parse.go` | `Parse`; the `builder` that walks goldmark's tree (`block` dispatcher, one method per node kind); spacing (`computeGaps`) |
| `inline.go` | Inline spans: `inlineRun.node` dispatcher, raw inline HTML (`<u>`, `<br>`), task boxes |
| `html.go` | HTML blocks, the `<div align="…">` wrapper |
| `math.go` | goldmark extension for `$$…$$` blocks and `$…$` inline maths |
| `highlight.go` | Syntax highlighting with chroma, language aliases |

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

### 6.4 `internal/render` — blocks to primitives

| File | What's in it |
|---|---|
| `render.go` | `Ctx`, `Prim`, `Layout`, `Theme`, list markers, paragraph/heading/code layouts, maths block, `mathPrims` |
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

### 6.5 `internal/pdfout` — primitives to PDF

| File | What's in it |
|---|---|
| `pdf.go` | `Write`, `exporter`, `placeBlock` (headings kept with the next lines), link targets |
| `paging.go` | `place` / `fillPage`: splitting a block across pages at `Layout.Breaks`; `segment` |
| `draw.go` | `drawPrim` dispatcher and one function per `Prim` kind |
| `fonts.go` | The embedded Go fonts, measuring, and **fallback glyphs** (⌘, ⇧, ✓ … drawn as outlines from the maths font) |

### 6.6 `internal/tex` — maths

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

### 6.7 `internal/imgcache` and `internal/youtube`

`imgcache.Fetch(src, done)` downloads in the background (the preview calls
it from `ui/images.go: Preview.image` and redraws in `imageArrived`);
`imgcache.Get(src)` downloads synchronously (the PDF calls it from
`pdfout/pdf.go: imageSize`). Only YouTube stills are allowed
(`youtube.IsThumbnail`). Failures aren't retried for a minute.

---

## 7. Recipes: where to add things

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
   screenshot scene (§8).

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

## 8. Finding bugs

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
  (split view, menus, dialogs, find bar, maths, videos …) to PNG files.
  Add a scene in `ui/screenshot_test.go` when you add something visual.
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
- **macOS builds** need `CC=clang` (the Makefile sets it); a Homebrew GCC
  can't compile Gio's Objective-C.

---

## 9. Code style

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
