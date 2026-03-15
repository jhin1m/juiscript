package system

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileManager handles filesystem operations with safety guarantees.
// Interface allows mocking in tests.
type FileManager interface {
	// WriteAtomic writes data to path atomically (temp file + rename).
	// This prevents partial writes that could corrupt config files.
	WriteAtomic(path string, data []byte, perm os.FileMode) error
	// Symlink creates a symbolic link pointing to target.
	Symlink(target, link string) error
	// RemoveSymlink removes only a symbolic link (safety check).
	RemoveSymlink(path string) error
	// Remove deletes a file or directory.
	Remove(path string) error
	// Exists checks if a path exists.
	Exists(path string) bool
	// ReadFile reads file contents.
	ReadFile(path string) ([]byte, error)
}

type fileOps struct{}

// NewFileManager creates a production file manager.
func NewFileManager() FileManager {
	return &fileOps{}
}

func (f *fileOps) WriteAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	// Ensure parent directory exists before writing
	// (e.g. /etc/php/8.3/fpm/pool.d/ may not exist yet)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	// Create temp file in the SAME directory so os.Rename is atomic
	// (rename across filesystems is not atomic)
	tmp, err := os.CreateTemp(dir, ".juiscript-tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	// Clean up temp file on any failure
	defer func() {
		if tmpPath != "" {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename to %s: %w", path, err)
	}

	tmpPath = "" // prevent deferred cleanup
	return nil
}

func (f *fileOps) Symlink(target, link string) error {
	// Remove existing symlink if present
	if info, err := os.Lstat(link); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			os.Remove(link)
		} else {
			return fmt.Errorf("path exists and is not a symlink: %s", link)
		}
	}

	return os.Symlink(target, link)
}

func (f *fileOps) RemoveSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // already gone
		}
		return err
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("not a symlink: %s", path)
	}

	return os.Remove(path)
}

func (f *fileOps) Remove(path string) error {
	return os.RemoveAll(path)
}

func (f *fileOps) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (f *fileOps) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
