//go:build js

package vfs

import (
	"bytes"
	"io"
	"sync"
	"time"
)

// file is a file kept in memory.
type file struct {
	data []byte
	mod  int64
}

var (
	mu    sync.Mutex
	files = map[string]file{}
)

// ReadFile returns the contents of a file.
func ReadFile(name string) ([]byte, error) {
	mu.Lock()
	defer mu.Unlock()
	f, ok := files[name]
	if !ok {
		return nil, ErrNotFound
	}
	return f.data, nil
}

// Open opens a file for reading.
func Open(name string) (io.ReadCloser, error) {
	data, err := ReadFile(name)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// ModTime returns when a file was written.
func ModTime(name string) (int64, error) {
	mu.Lock()
	defer mu.Unlock()
	f, ok := files[name]
	if !ok {
		return 0, ErrNotFound
	}
	return f.mod, nil
}

// WriteFile stores a file; it replaces the old contents in one step.
func WriteFile(name string, data []byte) error {
	mu.Lock()
	defer mu.Unlock()
	files[name] = file{data: bytes.Clone(data), mod: time.Now().UnixNano()}
	return nil
}
