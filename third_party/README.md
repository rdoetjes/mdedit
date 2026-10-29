# third_party

## gioui.org

A copy of [Gio](https://gioui.org) v0.10.3 with three small changes; `go.mod`
uses it through a `replace` directive. The first two make typing in long
documents fast, the third lets mdedit draw its own caret. The changes are
in `gioui.org.patch`:

1. `text/lru.go`: the shaper's caches hold 20000 entries instead of 1000.
   Gio caches one shaped line (paragraph) per entry, and the editor needs
   one for every different line of the document. With more than ~1000
   different lines the cache could never hold the document, so every key
   press shaped all of it again: HOW_IT_WORKS.md (~1550 different lines)
   took ~195 ms per key in the browser, ~52 ms on the desktop.
2. `text/gotext.go: document.append`: after appending a paragraph's lines,
   only those lines get their y offset computed, not all lines so far. That
   made laying out the editor's text quadratic in its number of lines.

3. `widget/editor.go`: `Editor.HideCaret` stops the editor drawing its thin
   caret; mdedit draws a steady yellow block caret instead
   (`internal/ui/caret.go`).

`go test ./widget/` reports one failure, `TestIndexPositionRunes`; Gio
v0.10.3 fails it unchanged as well.

With the first two, HOW_IT_WORKS.md types without lag in the browser; a document of
10,000 lines takes ~57 ms per key on the desktop and ~185 ms in the
browser, because Gio's editor still lays out (but no longer reshapes) the
whole text on each change.

To move to a newer Gio: copy the new module (`go env GOMODCACHE`), apply
the patch (`patch -p1 < ../gioui.org.patch` inside the copy), run
`go test ./text/` in it, and update the version in `go.mod` and here. Drop
a change once Gio itself has it.
