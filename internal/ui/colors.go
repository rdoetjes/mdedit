package ui

import "image/color"

// uiColors is the palette of the window around the preview: bars, menus,
// dialogs, the editor and the find bar. The preview itself uses a
// render.Theme.
type uiColors struct {
	toolbarBg   color.NRGBA // menu bar, toolbar, status bar, find bar
	buttonBg    color.NRGBA
	buttonFg    color.NRGBA // button labels, menu text, toolbar icons
	primaryBg   color.NRGBA // the default button of a dialog
	editorBg    color.NRGBA
	dividerCol  color.NRGBA // separators and outlines
	statusFg    color.NRGBA // secondary text: status bar, shortcuts
	selectColor color.NRGBA // editor selection
	borderColor color.NRGBA // text field frames
	errorColor  color.NRGBA
	highlightBg color.NRGBA // selected row in the file dialog
	cardBg      color.NRGBA // dialogs and drop-down menus
	backdrop    color.NRGBA // dims the window behind a dialog
	menuHover   color.NRGBA
	menuOpen    color.NRGBA // open menu title, hovered toolbar button
	popupShadow color.NRGBA
	matchBg     color.NRGBA // find: other matches
	currentBg   color.NRGBA // find: the selected match
}

func rgba(c uint32, a uint8) color.NRGBA {
	return color.NRGBA{R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c), A: a}
}

var lightColors = uiColors{
	toolbarBg:   rgba(0xf3f4f6, 0xff),
	buttonBg:    rgba(0xffffff, 0xff),
	buttonFg:    rgba(0x1f2328, 0xff),
	primaryBg:   rgba(0x1f883d, 0xff),
	editorBg:    rgba(0xfbfbfc, 0xff),
	dividerCol:  rgba(0xd1d9e0, 0xff),
	statusFg:    rgba(0x59636e, 0xff),
	selectColor: rgba(0x0969da, 0x40),
	borderColor: rgba(0xd1d9e0, 0xff),
	errorColor:  rgba(0xcf222e, 0xff),
	highlightBg: rgba(0xe8f0fe, 0xff),
	cardBg:      rgba(0xffffff, 0xff),
	backdrop:    rgba(0x000000, 0x66),
	menuHover:   rgba(0xe8f0fe, 0xff),
	menuOpen:    rgba(0xdde3ea, 0xff),
	popupShadow: rgba(0x000000, 0x22),
	matchBg:     rgba(0xffd33d, 0x66),
	currentBg:   rgba(0xff961f, 0x88),
}

// darkColors follow GitHub's dark mode.
var darkColors = uiColors{
	toolbarBg:   rgba(0x161b22, 0xff),
	buttonBg:    rgba(0x21262d, 0xff),
	buttonFg:    rgba(0xe6edf3, 0xff),
	primaryBg:   rgba(0x238636, 0xff),
	editorBg:    rgba(0x0d1117, 0xff),
	dividerCol:  rgba(0x30363d, 0xff),
	statusFg:    rgba(0x9198a1, 0xff),
	selectColor: rgba(0x388bfd, 0x55),
	borderColor: rgba(0x3d444d, 0xff),
	errorColor:  rgba(0xf85149, 0xff),
	highlightBg: rgba(0x1c2b41, 0xff),
	cardBg:      rgba(0x161b22, 0xff),
	backdrop:    rgba(0x000000, 0x99),
	menuHover:   rgba(0x1c2b41, 0xff),
	menuOpen:    rgba(0x30363d, 0xff),
	popupShadow: rgba(0x000000, 0x66),
	matchBg:     rgba(0xbb8009, 0x77),
	currentBg:   rgba(0xdb6d28, 0x99),
}

// col is the current palette. The UI runs on a single goroutine, so the
// theme switch can simply replace it.
var col = lightColors
