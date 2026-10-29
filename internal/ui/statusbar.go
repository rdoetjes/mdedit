package ui

import (
	"fmt"
	"image"
	"strings"
	"time"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget/material"
)

// statusTimeout is how long a status message stays visible.
const statusTimeout = 8 * time.Second

const shortcutHelp = "Ctrl+O open · Ctrl+S save · Ctrl+E export PDF · Ctrl+/ switch view"

func (a *App) statusBar(gtx layout.Context) layout.Dimensions {
	return layout.Background{}.Layout(gtx, barWithTopLine, a.statusContent)
}

func barWithTopLine(gtx layout.Context) layout.Dimensions {
	sz := gtx.Constraints.Min
	fillRect(gtx, col.toolbarBg, image.Rectangle{Max: sz})
	fillRect(gtx, col.dividerCol, image.Rect(0, 0, sz.X, 1))
	return layout.Dimensions{Size: sz}
}

func (a *App) statusContent(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.Inset{Top: 4, Bottom: 4, Left: 10, Right: 10}.Layout(gtx, a.statusRow)
}

func (a *App) statusRow(gtx layout.Context) layout.Dimensions {
	return layout.Flex{}.Layout(gtx,
		layout.Flexed(1, a.statusMessage),
		layout.Rigid(a.caretInfo),
	)
}

func (a *App) statusMessage(gtx layout.Context) layout.Dimensions {
	l := material.Caption(a.th, a.statusText(gtx))
	l.Color = col.statusFg
	l.MaxLines = 1
	return l.Layout(gtx)
}

// statusText is the toolbar hint, a recent status message, or the main
// shortcuts.
func (a *App) statusText(gtx layout.Context) string {
	if a.hint != "" {
		return a.hint
	}
	if a.status != "" && time.Since(a.statusTime) < statusTimeout {
		gtx.Execute(op.InvalidateCmd{At: a.statusTime.Add(statusTimeout)})
		return a.status
	}
	if key.ModShortcut == key.ModCommand {
		return strings.ReplaceAll(shortcutHelp, "Ctrl+", "⌘")
	}
	return shortcutHelp
}

func (a *App) caretInfo(gtx layout.Context) layout.Dimensions {
	row, column := a.caretLine()
	l := material.Caption(a.th, fmt.Sprintf("Ln %d, Col %d · %d words", row+1, column+1, a.words))
	l.Color = col.statusFg
	return l.Layout(gtx)
}
