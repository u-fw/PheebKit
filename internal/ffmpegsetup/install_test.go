package ffmpegsetup

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for name, value := range files {
		file, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestInstallerRetriesAndRetainsNotices(t *testing.T) {
	data := zipBytes(t, map[string]string{"ffmpeg/bin/ffmpeg.exe": "binary", "ffmpeg/bin/ffprobe.exe": "probe", "ffmpeg/LICENSE": "upstream", "ffmpeg/README.txt": "source references"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/failed" {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Write(data)
	}))
	defer server.Close()
	root := t.TempDir()
	installer := &Installer{Client: server.Client()}
	result, err := installer.Install(context.Background(), root, []Package{{ID: "test", Version: "1", Artifacts: []Artifact{{Name: "ffmpeg.zip", Format: "zip", SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), URLs: []string{server.URL + "/failed"}}}}, {ID: "fallback", Version: "2", Artifacts: []Artifact{{Name: "ffmpeg.zip", Format: "zip", SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), URLs: []string{server.URL + "/ok"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "2" {
		t.Fatal("fallback package not recorded")
	}
	if _, err := os.Stat(result.FFprobe); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(result.Directory, "ffmpeg", "LICENSE")); err != nil {
		t.Fatal("license lost", err)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 1 || strings.HasPrefix(entries[0].Name(), ".download-") {
		t.Fatal("partial directory leaked")
	}
}

func TestInstallerRejectsHashMismatchWithoutPublishing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("untrusted")) }))
	defer server.Close()
	root := t.TempDir()
	installer := &Installer{Client: server.Client()}
	_, err := installer.Install(context.Background(), root, []Package{{ID: "test", Artifacts: []Artifact{{Name: "ffmpeg.exe", Format: "plain", SHA256: strings.Repeat("0", 64), URLs: []string{server.URL}}}}})
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("hash mismatch accepted: %v", err)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("failed artifacts published")
	}
}

func TestExtractRejectsTraversal(t *testing.T) {
	for _, files := range []map[string]string{{"../escape.exe": "bad"}, {"C:/escape.exe": "bad"}, {"folder\\escape.exe": "bad"}} {
		directory := t.TempDir()
		archive := filepath.Join(directory, "input.zip")
		os.WriteFile(archive, zipBytes(t, files), 0600)
		if err := extract(context.Background(), archive, directory, Artifact{Format: "zip"}); err == nil {
			t.Fatal("unsafe zip path accepted", files)
		}
	}
}

func TestInstallerCancelsMidTransfer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("partial"))
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	installer := &Installer{Client: server.Client()}
	directory := t.TempDir()
	done := make(chan error, 1)
	go func() {
		done <- installer.download(ctx, server.URL, filepath.Join(directory, "partial"), Artifact{SHA256: strings.Repeat("0", 64)}, "test")
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled transfer succeeded")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("transfer ignored cancellation")
	}
}

func TestInstallerRejectsUntrustedRedirect(t *testing.T) {
	client := New(nil).Client
	req, _ := http.NewRequest(http.MethodGet, "https://evil.example/file", nil)
	if err := client.CheckRedirect(req, nil); err == nil {
		t.Fatal("redirect policy accepted unknown host")
	}
}

func TestCancelledInstallLeavesNoPartialFiles(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	root := t.TempDir()
	_, err := New(nil).Install(ctx, root, []Package{{ID: "unused"}})
	if err != context.Canceled {
		t.Fatalf("unexpected cancellation: %v", err)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("cancelled install leaked files")
	}
}

func TestCatalogAndRedirectPolicy(t *testing.T) {
	for _, source := range []string{"auto", "stable", "cn"} {
		packages, err := Catalog(source)
		if err != nil {
			t.Fatal(err)
		}
		for _, pkg := range packages {
			for _, artifact := range pkg.Artifacts {
				if len(artifact.SHA256) != 64 {
					t.Fatal("unpinned artifact")
				}
				for _, raw := range artifact.URLs {
					if !trustedURL(raw) {
						t.Fatal("untrusted catalog URL", raw)
					}
				}
			}
		}
	}
	for _, raw := range []string{"http://github.com/file", "https://github.com.evil.example/file", "https://github.com:8443/file", "https://user:password@github.com/file", "https://127.0.0.1/file"} {
		if trustedURL(raw) {
			t.Fatal("unsafe URL accepted", raw)
		}
	}
	if _, err := Catalog("https://example.com"); err == nil {
		t.Fatal("arbitrary source accepted")
	}
}
