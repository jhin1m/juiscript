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

func TestRemove(t *testing.T) {
	fm := NewFileManager()
	dir := t.TempDir()
	path := filepath.Join(dir, "to-delete.txt")
	os.WriteFile(path, []byte("delete me"), 0644)

	if err := fm.Remove(path); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if fm.Exists(path) {
		t.Error("file should be removed")
	}
}

func TestRemoveDir(t *testing.T) {
	fm := NewFileManager()
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	os.MkdirAll(subdir, 0755)
	os.WriteFile(filepath.Join(subdir, "file.txt"), []byte("data"), 0644)

	if err := fm.Remove(subdir); err != nil {
		t.Fatalf("Remove dir failed: %v", err)
	}
	if fm.Exists(subdir) {
		t.Error("directory should be removed")
	}
}

func TestReadFile(t *testing.T) {
	fm := NewFileManager()
	dir := t.TempDir()
	path := filepath.Join(dir, "read-test.txt")
	content := []byte("hello world")
	os.WriteFile(path, content, 0644)

	got, err := fm.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("ReadFile = %q, want %q", got, content)
	}
}

func TestReadFile_NotExist(t *testing.T) {
	fm := NewFileManager()
	_, err := fm.ReadFile("/tmp/definitely-does-not-exist-juiscript-read")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestWriteAtomic_Permissions(t *testing.T) {
	fm := NewFileManager()
	dir := t.TempDir()
	path := filepath.Join(dir, "perm-test.conf")

	if err := fm.WriteAtomic(path, []byte("secret"), 0600); err != nil {
		t.Fatalf("WriteAtomic failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestSymlink_OverwriteExisting(t *testing.T) {
	fm := NewFileManager()
	dir := t.TempDir()

	target1 := filepath.Join(dir, "target1.conf")
	target2 := filepath.Join(dir, "target2.conf")
	os.WriteFile(target1, []byte("v1"), 0644)
	os.WriteFile(target2, []byte("v2"), 0644)

	link := filepath.Join(dir, "link.conf")

	// Create first symlink
	if err := fm.Symlink(target1, link); err != nil {
		t.Fatalf("first Symlink failed: %v", err)
	}

	// Overwrite with second target
	if err := fm.Symlink(target2, link); err != nil {
		t.Fatalf("overwrite Symlink failed: %v", err)
	}

	// Verify it now points to target2
	resolved, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("Readlink failed: %v", err)
	}
	if resolved != target2 {
		t.Errorf("link points to %q, want %q", resolved, target2)
	}
}
