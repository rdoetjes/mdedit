# MDEdit

A Markdown editor with a live, GitHub-styled preview and PDF export, written
in Go with the [Gio](https://gioui.org) UI toolkit. Runs on Linux (X11 and Wayland), FreeBSD, macOS and Windows.

![Screenshot](./screenshot.png)

<!-- toc -->

- [MDEdit](#mdedit)
  - [Building](#building)
  - [Usage](#usage)
    - [Maths](#maths)
    - [Fenced blocks (code blocks)](#fenced-blocks-code-blocks)
  - [Supported Markdown](#supported-markdown)
  - [Layout](#layout)

<!-- /toc -->

## Building

    make deps     # lists the system packages to install for your OS
    make          # builds bin/mdedit
    make install  # binary, menu entry and icons under /usr/local (PREFIX=... to change)

`make install` (run with sudo) also adds MDEdit to the application menu on
Linux and FreeBSD desktops: `share/applications/mdedit.desktop` (categories
Development, Office and Science; it opens Markdown files) and the icon in
all standard sizes under `share/icons/hicolor`. `make desktop` only writes
`bin/mdedit.desktop`; `make icons` redraws the icons from `tools/mkicon`.
For a per-user install without root, `make-desktop.sh` is still there.

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
| Ctrl+F / Ctrl+R        | Find / Replace            | Ctrl+G / Ctrl+Shift+G | Next / previous match |

Find (Ctrl+F) and Replace (Ctrl+R) open a bar above the editor; the
selected text becomes the search text. All matches are highlighted, the
counter shows "3 of 12", Enter and Shift+Enter go to the next and previous
match (wrapping around), and Escape closes the bar. Matching ignores case
unless Match case is ticked. Replace replaces the selected match and moves
to the next; Replace All replaces every match in one undoable step.

Pressing Enter in a list item or quote starts the next item (numbers count
up); Enter on an empty item ends the list. Shift+Enter inserts a plain line
break.

Tab inserts spaces to the next 4-column stop; on list items (or several
selected lines) Tab and Shift+Tab indent and outdent, nesting list items.
Tab characters in opened or pasted text are converted to spaces, which is
how CommonMark reads them anyway (Gio's text engine can't display tabs).

Insert → Table of Contents adds a nested list of links to all headings,
between `<!-- toc -->` and `<!-- /toc -->` markers (hidden when rendered,
and the convention markdown-toc and VS Code use). Right-click in the editor
or preview and choose Update Table of Contents to regenerate it after
editing headings; the right-click menu also has Cut, Copy and Paste.

### YouTube videos

Videos use GitHub's notation, a link around the video's still:

    [![Video Title](https://img.youtube.com/vi/VIDEO_ID/maxresdefault.jpg)](https://www.youtube.com/watch?v=VIDEO_ID)

Insert → YouTube Video… (or the toolbar button) asks for the video ID — a
full YouTube link works too — and an optional title. The preview shows the
still at 66% of the width with a play button; clicking it opens the video in
the browser. PDFs show just the still, linked to the video.

Stills are downloaded in the background and cached in `~/.cache/mdedit/images`
(`~/Library/Caches/mdedit/images` on macOS). Only YouTube stills are fetched,
never other remote images; when a video has no full-resolution still,
YouTube's standard one is used.

Example me covering the Trooper: 
<div align="center">

[![The Trooper](https://img.youtube.com/vi/XIiJa9yDecU/maxresdefault.jpg)](https://www.youtube.com/watch?v=XIiJa9yDecU)

</div>

### Maths

Display maths goes between `$$` lines (or ` ```math ` fences, as on GitHub);
inline maths between single dollars: `$e^{i\pi} + 1 = 0$`. As in Pandoc and
Typora, `$` followed by a space, or a closing `$` before a digit, isn't
maths, so `$5 and $10` stays text; write `\$` for a literal dollar sign.

mdedit typesets maths itself (no JavaScript, no network) with the STIX Two
Math font (SIL Open Font License, embedded): fractions, roots, sub- and
superscripts, large operators with limits, stretchy `\left … \right`
delimiters, accents, Greek and symbols, `\text`, `\mathbb` / `\mathcal` /
`\mathbf` …, and the `matrix` / `pmatrix` / `bmatrix` / `vmatrix`, `cases`,
`aligned` and `array` environments. Lines separated by `\\` (with `&`
alignment points) work without an environment too. Unknown commands are
shown in red. In PDFs, maths is drawn as vector outlines, so it isn't
selectable as text.

Insert → Math Block (Ctrl+Shift+M) and Format → Inline Math add the markup.

Example 1:
$$
E=mc^2
$$
Example 2:
$$
\sqrt{a}=b \qquad \text{iff} \quad b^2=a \
$$
Example 3:
$$
\sqrt{e}\, \sqrt{T}\,\sqrt{T^2_1}
$$

### Fenced blocks (code blocks)

Fenced code blocks are highlighted by the language after the fence
(` ```go `, ` ```bash `, ` ```C# `, ` ```c++ `, ` ```yaml `, ` ```{.python} `
…), covering about 250 languages and their usual aliases.

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
    internal/edit      formatting commands as pure functions on text + selection
    internal/tex       TeX maths typesetter (embedded STIX Two Math)
    internal/youtube   YouTube URLs and video IDs
    internal/imgcache  downloads and caches YouTube stills
    internal/lint      code-style test: no anonymous functions, ≤4 indent levels, ≤40-line functions
    tools/mkicon       draws the application icon

The preview and the PDF share `internal/render`, so both use the same layout.

See [HOW_IT_WORKS.md](HOW_IT_WORKS.md) for how the layers fit together, how
input flows through the code, where to add features and where to look for
bugs.
