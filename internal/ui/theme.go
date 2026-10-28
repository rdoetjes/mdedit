package ui

import (
	"encoding/json"

	"gioui.org/layout"
	"gioui.org/widget/material"
	"gioui.org/x/pref/theme"

	"mdedit/internal/render"
)

// settings are the preferences kept between runs.
type settings struct {
	Theme      string   `json:"theme,omitempty"` // "light" or "dark"; empty follows the system
	NoSpelling bool     `json:"noSpelling,omitempty"`
	Words      []string `json:"words,omitempty"` // added to the dictionary
}

func loadSettings() settings {
	var s settings
	if data, err := readSettings(); err == nil {
		json.Unmarshal(data, &s) // a broken file just means defaults
	}
	return s
}

func (s settings) save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return writeSettings(data)
}

// startDark decides the theme at start-up: the saved choice, or else the
// system's dark mode where it can be detected (macOS, Windows).
func startDark(s settings) bool {
	switch s.Theme {
	case "dark":
		return true
	case "light":
		return false
	}
	dark, err := theme.IsDarkMode()
	return err == nil && dark
}

// setDark switches the whole window — bars, dialogs, editor and preview —
// to the dark or light theme.
func (a *App) setDark(dark bool) {
	a.dark = dark
	col, a.th.Palette = lightColors, lightPalette
	preview := &render.Light
	label := "Dark Theme"
	if dark {
		col, a.th.Palette = darkColors, darkPalette
		preview = &render.Dark
		label = "Light Theme"
	}
	a.preview.SetTheme(preview)
	a.cmds.theme.label = label
}

func themeName(dark bool) string {
	if dark {
		return "dark"
	}
	return "light"
}

// Gio's own palette colours text, text fields, check boxes and scroll bars.
var (
	lightPalette = material.NewTheme().Palette // Gio's defaults, as before themes existed
	darkPalette  = material.Palette{Fg: rgba(0xe6edf3, 0xff), Bg: rgba(0x0d1117, 0xff),
		ContrastBg: rgba(0x4493f8, 0xff), ContrastFg: rgba(0xffffff, 0xff)}
)

// cmdToggleTheme switches theme and remembers the choice.
func (a *App) cmdToggleTheme(layout.Context) {
	a.setDark(!a.dark)
	s := loadSettings()
	s.Theme = themeName(a.dark)
	if err := s.save(); err != nil {
		a.setStatus("Could not save the theme setting: %v", err)
	}
}
