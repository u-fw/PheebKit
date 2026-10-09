package windows

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublicationNeverReplacesExistingOutput(t *testing.T) {
	dir := t.TempDir()
	from, to := filepath.Join(dir, "scratch.mp4"), filepath.Join(dir, "output.mp4")
	if err := os.WriteFile(from, []byte("encoded media"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, []byte("other process output"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := publishFile(from, to); err == nil {
		t.Fatal("publication overwrote an existing destination")
	}
	data, err := os.ReadFile(to)
	if err != nil || string(data) != "other process output" {
		t.Fatalf("destination was changed: %q %v", data, err)
	}
}

func TestScratchCleanupRejectsOutsideParent(t *testing.T) {
	parent := t.TempDir()
	other := t.TempDir()
	if err := cleanupScratch(other, parent); err == nil {
		t.Fatal("cleanup accepted an unrelated directory")
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("cleanup affected unrelated directory: %v", err)
	}
}
