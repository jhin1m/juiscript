package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomic(t *testing.T) {
	fm := NewFileManager()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	data := []byte("server { listen 80; }")
	if err := fm.WriteAtomic(path, data, 0644); err != nil {
		t.Fatalf("WriteAtomic failed: %v", err)
	}

	// Verify contents
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("content mismatch: got %q", got)
	}

	// Verify no temp files left behind
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected 1 file, got %d", len(entries))
	}
}

func TestSymlinkAndRemove(t *testing.T) {
	fm := NewFileManager()
	dir := t.TempDir()

	target := filepath.Join(dir, "target.conf")
	os.WriteFile(target, []byte("test"), 0644)

	link := filepath.Join(dir, "link.conf")

	// Create symlink
	if err := fm.Symlink(target, link); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	if !fm.Exists(link) {
		t.Error("symlink should exist")
	}

	// Remove symlink
	if err := fm.RemoveSymlink(link); err != nil {
		t.Fatalf("RemoveSymlink failed: %v", err)
	}

	if fm.Exists(link) {
		t.Error("symlink should be removed")
	}

	// Target should still exist
	if !fm.Exists(target) {
		t.Error("target should still exist after removing symlink")
	}
}

func TestRemoveSymlinkNonSymlink(t *testing.T) {
	fm := NewFileManager()
	dir := t.TempDir()

	// Regular file, not a symlink
	path := filepath.Join(dir, "regular.conf")
	os.WriteFile(path, []byte("test"), 0644)

	err := fm.RemoveSymlink(path)
	if err == nil {
		t.Error("expected error when removing non-symlink")
	}
}

func TestExistsNonExistent(t *testing.T) {
	fm := NewFileManager()
	if fm.Exists("/tmp/definitely-does-not-exist-juiscript") {
		t.Error("non-existent path should return false")
	}
}
