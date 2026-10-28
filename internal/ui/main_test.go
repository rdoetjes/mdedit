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
	// Start light whatever the system's appearance, so tests and
	// screenshots don't depend on it.
	if err := (settings{Theme: "light"}).save(); err != nil {
		panic(err)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
