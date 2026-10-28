package vfs

import (
	"io"
	"path/filepath"
	"testing"
)

func TestWriteReadOpen(t *testing.T) {
	name := filepath.Join(t.TempDir(), "sub", "a.txt")
	if _, err := ModTime(name); err == nil {
		t.Fatal("a missing file should have no modification time")
	}
	if err := WriteFile(name, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	if data, err := ReadFile(name); err != nil || string(data) != "hello" {
		t.Fatalf("ReadFile = %q, %v", data, err)
	}
	if _, err := ModTime(name); err != nil {
		t.Fatal(err)
	}
	r, err := Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if data, _ := io.ReadAll(r); string(data) != "hello" {
		t.Fatalf("Open read %q", data)
	}
}

func TestFolderIsNotAFile(t *testing.T) {
	if _, err := ModTime(t.TempDir()); err == nil {
		t.Fatal("a folder should count as missing")
	}
}
