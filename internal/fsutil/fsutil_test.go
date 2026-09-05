package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.bin")

	if err := WriteFileAtomic(path, []byte("hello"), 0600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil || string(data) != "hello" {
		t.Fatalf("content mismatch: %q err=%v", data, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("perm = %v, want 0600", got)
	}

	// No temp leftovers in the target directory.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected only the target file, got %d entries", len(entries))
	}

	// Overwrite keeps content fresh.
	if err := WriteFileAtomic(path, []byte("second"), 0600); err != nil {
		t.Fatalf("overwrite failed: %v", err)
	}
	data, _ = os.ReadFile(path)
	if string(data) != "second" {
		t.Errorf("overwrite content = %q", data)
	}
}

// Unwritable target directory must return an error, not create anything.
func TestWriteFileAtomicUnwritableDir(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "file-not-dir") // exists as a file → mkdir fails
	if err := os.WriteFile(blocked, []byte("x"), 0600); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	if err := WriteFileAtomic(filepath.Join(blocked, "inner", "out.bin"), []byte("y"), 0600); err == nil {
		t.Error("expected an error when the target directory cannot be created")
	}
}
