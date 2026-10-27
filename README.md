# mdedit

A Markdown editor with a live, GitHub-styled preview and PDF export, written
in Go with the [Gio](https://gioui.org) UI toolkit. Runs on Linux (X11 and
Wayland), FreeBSD, macOS and Windows.

## Building

    make deps     # lists the system packages to install for your OS
    make          # builds bin/mdedit
    make install  # installs to /usr/local/bin (PREFIX=... to change)

On FreeBSD both `make` and `gmake` work. The result is a single executable
containing all Go code and fonts; it only loads the system's windowing and
GPU libraries at run time.

## Usage

    mdedit [file.md]                   # edit (a missing file is created on save)
    mdedit -pdf file.md [-o out.pdf]   # export without opening a window

All commands are in the menu bar (File, Edit, Format, Insert, View, Help);
the toolbar holds the common ones: bold, italic, underline, alignment,
numbered/bullet lists and table. Main shortcuts (⌘ instead of Ctrl on macOS):

| Shortcut               | Action                    | Shortcut          | Action           |
|------------------------|---------------------------|-------------------|------------------|
| Ctrl+N / Ctrl+O        | New / Open                | Ctrl+B / I / U    | Bold / Italic / Underline |
| Ctrl+S / Ctrl+Shift+S  | Save / Save as            | Ctrl+Shift+L/E/R  | Align left / center / right |
| Ctrl+E                 | Export PDF                | Ctrl+1 … 6, Ctrl+0 | Heading 1–6 / paragraph |
| Ctrl+/                 | Cycle split/editor/preview | Ctrl+T           | Insert table     |
| Ctrl+Q                 | Quit                      | Ctrl+K / Ctrl+Shift+K | Link / code block |

Pressing Enter in a list item or quote starts the next item (numbers count
up); Enter on an empty item ends the list. Shift+Enter inserts a plain line
break.

On macOS the usual ⌘ navigation works in the editor: ⌘←/⌘→ to the start/end
of the line, ⌘↑/⌘↓ to the start/end of the document (add ⇧ to select), and
⌘⌫ to delete to the start of the line. Home/End (fn+←/→) work everywhere.

Markdown has no underline or alignment syntax, so mdedit uses the HTML that
GitHub and Typora render: `<u>text</u>`, and a `<div align="center">` wrapper
with blank lines inside so the content is still parsed as Markdown.

The divider between editor and preview can be dragged. The preview follows
the caret. Clicking a `#anchor` link jumps to the heading, relative `.md`
links open in the editor, and other links open in the system browser.

If the window is closed with unsaved changes, they are written to
`~/.cache/mdedit/` (`~/Library/Caches/mdedit/` on macOS) and the path is
printed on stderr.

## Supported Markdown

CommonMark plus GitHub extensions: headings (ATX and Setext), paragraphs and
hard line breaks, emphasis, strong, ~~strikethrough~~, inline code, fenced and
indented code blocks with syntax highlighting (via
[chroma](https://github.com/alecthomas/chroma), ~250 languages), block quotes
(nested, with any content), ordered/unordered/nested lists, task lists,
tables with column alignment, horizontal rules, links, autolinks, local
images (PNG, JPEG, GIF) and backslash escapes/entities.

## Layout

    cmd/mdedit         entry point
    internal/mdmodel   Markdown → flat list of styled blocks (goldmark), highlighting
    internal/render    device-independent layout: blocks → positioned primitives
    internal/ui        Gio window, preview painter, file dialog
    internal/pdfout    PDF painter with page breaking (fpdf)

The preview and the PDF share `internal/render`, so both use the same layout.
