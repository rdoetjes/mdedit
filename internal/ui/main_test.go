//go:build !js

package ui

import (
	"os"
	"testing"
)

// TestMain keeps the tests away from the user's settings and the system
// appearance, so a theme choice doesn't change what the tests see.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "mdedit-settings")
	if err != nil {
		panic(err)
	}
	configDir = dir
	resetSettings()
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// resetSettings starts a test from the default settings, but light
// whatever the system's appearance, so tests and screenshots don't depend
// on it or on what earlier tests chose.
func resetSettings() {
	if err := (settings{Theme: "light"}).save(); err != nil {
		panic(err)
	}
}
