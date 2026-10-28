//go:build !js

package vfs

import (
	"io"
	"os"
	"path/filepath"
)

// ReadFile returns the contents of a file.
func ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }

// Open opens a file for reading.
func Open(name string) (io.ReadCloser, error) { return os.Open(name) }

// ModTime returns a file's modification time; folders count as missing.
func ModTime(name string) (int64, error) {
	st, err := os.Stat(name)
	if err != nil {
		return 0, err
	}
	if st.IsDir() {
		return 0, ErrNotFound
	}
	return st.ModTime().UnixNano(), nil
}

// WriteFile writes a file atomically, so readers never see half of it.
func WriteFile(name string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	tmp := name + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, name)
}
