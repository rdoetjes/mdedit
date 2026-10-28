// Package vfs is the file storage the image code reads and writes: the
// disk on the desktop, memory in the browser, where WebAssembly programs
// have no file system. The editor's own documents don't go through it;
// opening and saving them is platform-specific (package ui).
//
// The functions are the same on both platforms:
//
//	ReadFile(name) ([]byte, error)
//	WriteFile(name, data) error   // atomic; creates missing folders
//	Open(name) (io.ReadCloser, error)
//	ModTime(name) (int64, error)  // nanoseconds; an error for missing files and folders
package vfs

import "errors"

// ErrNotFound is returned for a file that doesn't exist.
var ErrNotFound = errors.New("file not found")
