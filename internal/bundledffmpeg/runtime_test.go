package bundledffmpeg

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runtimeArchive(t *testing.T, mutate func(map[string][]byte)) []byte {
	t.Helper()
	ffmpeg, ffprobe := []byte("ffmpeg.exe"), []byte("ffprobe.exe")
	metadata := Metadata{
		SchemaVersion:       1,
		Provider:            "test",
		Package:             "test-1",
		Version:             "1.0",
		VersionLine:         "ffmpeg version test",
		License:             "test",
		CorrespondingSource: "https://example.test/source",
		ArchiveURL:          "https://example.test/archive.zip",
		ArchiveSHA256:       strings.Repeat("1", 64),
		FFmpegSHA256:        digest(ffmpeg),
		FFprobeSHA256:       digest(ffprobe),
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"BUNDLED_FFMPEG.json": metadataJSON,
		"LICENSE.txt":         []byte("license"),
		"README.txt":          []byte("readme"),
		"bin/ffmpeg.exe":      ffmpeg,
		"bin/ffprobe.exe":     ffprobe,
		"presets/test.txt":    []byte("preset"),
	}
	if mutate != nil {
		mutate(files)
	}
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for name, data := range files {
		file, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestPreparePayloadExtractsAndReusesRuntime(t *testing.T) {
	payload := runtimeArchive(t, nil)
	root := t.TempDir()
	expected := digest(payload)
	first, err := preparePayload(payload, expected, "1.0-test", root)
	if err != nil {
		t.Fatal(err)
	}
	if first.Metadata.Version != "1.0" || first.PayloadSHA256 != expected {
		t.Fatalf("unexpected runtime: %+v", first)
	}
	for _, path := range []string{first.FFmpeg, first.FFprobe, filepath.Join(first.Directory, "LICENSE.txt")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	second, err := preparePayload(payload, expected, "1.0-test", root)
	if err != nil {
		t.Fatal(err)
	}
	if second.Directory != first.Directory {
		t.Fatalf("runtime was extracted twice: %s and %s", first.Directory, second.Directory)
	}
}

func TestPreparePayloadRejectsDigestMismatch(t *testing.T) {
	payload := runtimeArchive(t, nil)
	_, err := preparePayload(payload, strings.Repeat("0", 64), "1.0-test", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("digest mismatch accepted: %v", err)
	}
}

func TestPreparePayloadRejectsUnsafeAndCorruptData(t *testing.T) {
	payload := runtimeArchive(t, func(files map[string][]byte) {
		delete(files, "bin/ffprobe.exe")
	})
	_, err := preparePayload(payload, digest(payload), "1.0-test", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "ffprobe") {
		t.Fatalf("incomplete runtime accepted: %v", err)
	}

	unsafe := runtimeArchive(t, func(files map[string][]byte) {
		files["../escape"] = []byte("bad")
	})
	_, err = preparePayload(unsafe, digest(unsafe), "1.0-test", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "unsafe path") {
		t.Fatalf("unsafe runtime accepted: %v", err)
	}

	_, err = preparePayload([]byte("not zip"), digest([]byte("not zip")), "1.0-test", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "ZIP") {
		t.Fatalf("corrupt runtime accepted: %v", err)
	}
}

func TestPreparePayloadReplacesCorruptCache(t *testing.T) {
	payload := runtimeArchive(t, nil)
	root := t.TempDir()
	expected := digest(payload)
	final := filepath.Join(root, "ffmpeg-1.0-test-"+expected[:12])
	if err := os.MkdirAll(final, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(final, "BUNDLED_FFMPEG.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	runtime, err := preparePayload(payload, expected, "1.0-test", root)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.Directory != final {
		t.Fatalf("unexpected replacement directory: %s", runtime.Directory)
	}
}

func TestPreparePayloadRefusesSymlinkReplacement(t *testing.T) {
	payload := runtimeArchive(t, nil)
	root := t.TempDir()
	expected := digest(payload)
	outside := t.TempDir()
	final := filepath.Join(root, "ffmpeg-1.0-test-"+expected[:12])
	if err := os.Symlink(outside, final); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err := preparePayload(payload, expected, "1.0-test", root)
	if err == nil || !strings.Contains(err.Error(), "non-directory") {
		t.Fatalf("symlink cache entry was replaced: %v", err)
	}
}

func TestStandardBuildReportsExternalDistribution(t *testing.T) {
	if len(embeddedPayload()) != 0 {
		t.Skip("embedded build")
	}
	description := DescribeBuild()
	if description.Kind != "external" {
		t.Fatalf("unexpected standard build description: %+v", description)
	}
	if _, err := Resolve(); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("standard build unexpectedly resolves an embedded runtime: %v", err)
	}
}
