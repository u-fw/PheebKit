package sourcefile

import (
	"bytes"
	"crypto/sha256"
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
	// Outside the old three samples: full small-file hashing must catch it even
	// when the filesystem's coarse change time does not advance.
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

func TestSmallContentHashAndLargeReadBudget(t *testing.T) {
	data := bytes.Repeat([]byte{29}, 1024*1024)
	digest := func(data []byte, legacy bool) string {
		hash := sha256.New()
		if err := writeContentSamples(hash, bytes.NewReader(data), int64(len(data)), legacy); err != nil {
			t.Fatal(err)
		}
		return string(hash.Sum(nil))
	}
	before, oldBefore := digest(data, false), digest(data, true)
	data[128*1024] = 30
	if before == digest(data, false) {
		t.Fatal("small-file content mutation was missed")
	}
	if oldBefore != digest(data, true) {
		t.Fatal("fixture must be outside the legacy samples")
	}
	large := &countedReaderAt{Reader: bytes.NewReader(make([]byte, fullHashLimit+1))}
	if err := writeContentSamples(sha256.New(), large, fullHashLimit+1, false); err != nil {
		t.Fatal(err)
	}
	if large.read != 192*1024 {
		t.Fatalf("large-file read budget changed: %d", large.read)
	}
}

type countedReaderAt struct {
	*bytes.Reader
	read int
}

func (r *countedReaderAt) ReadAt(p []byte, offset int64) (int, error) {
	n, err := r.Reader.ReadAt(p, offset)
	r.read += n
	return n, err
}

func TestLegacyFingerprintCompatibility(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.mp4")
	if err := os.WriteFile(path, bytes.Repeat([]byte{7}, 1024*1024), 0600); err != nil {
		t.Fatal(err)
	}
	old, err := fingerprint(path, true)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Fingerprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if old == current {
		t.Fatal("new probes must issue version-2 fingerprints")
	}
	if err := Verify(path, old); err != nil {
		t.Fatalf("unchanged historical reference rejected: %v", err)
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
