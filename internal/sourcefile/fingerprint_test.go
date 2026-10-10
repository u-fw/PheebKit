package sourcefile

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFingerprintRemainsStableAcrossReads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "媒体.mp4")
	if err := os.WriteFile(path, bytes.Repeat([]byte{17}, 256*1024), 0600); err != nil {
		t.Fatal(err)
	}
	fingerprint, err := Fingerprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.ReadFile(path); err != nil {
		t.Fatal(err)
	}
	if err := Verify(path, fingerprint); err != nil {
		t.Fatalf("reading an unchanged video invalidated it: %v", err)
	}
}

func TestSameSizeEditWithRestoredModificationTimeIsDetected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "same-size.mp4")
	data := bytes.Repeat([]byte{29}, 1024*1024)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	modified := time.Unix(1_600_000_000, 0)
	if err := os.Chtimes(path, modified, modified); err != nil {
		t.Fatal(err)
	}
	fingerprint, err := Fingerprint(path)
	if err != nil {
		t.Fatal(err)
	}
	// Outside all three content samples: filesystem change time must catch it.
	file, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteAt([]byte{30}, 128*1024); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modified, modified); err != nil {
		t.Fatal(err)
	}
	if err := Verify(path, fingerprint); !errors.Is(err, ErrChanged) {
		t.Fatalf("same-size edit with restored mtime was accepted: %v", err)
	}
}

func TestSamePathReplacementWithIdenticalContentIsDetected(t *testing.T) {
	directory := t.TempDir()
	path, replacement := filepath.Join(directory, "input.mp4"), filepath.Join(directory, "replacement.mp4")
	modified := time.Unix(1_600_000_000, 0)
	for _, file := range []string{path, replacement} {
		if err := os.WriteFile(file, []byte("identical bytes"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(file, modified, modified); err != nil {
			t.Fatal(err)
		}
	}
	fingerprint, err := Fingerprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	if err := Verify(path, fingerprint); !errors.Is(err, ErrChanged) {
		t.Fatalf("a replacement file reused its predecessor's fingerprint: %v", err)
	}
}
