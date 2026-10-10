package ffmpegsetup

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCustomZIPValidationAndPinnedMirror(t *testing.T) {
	mirror, err := CustomZIP("https://downloads.example/file.zip?token=private", "", false)
	if err != nil {
		t.Fatal(err)
	}
	stable, _ := Catalog("stable")
	if mirror.Artifacts[0].SHA256 != stable[0].Artifacts[0].SHA256 || mirror.Artifacts[0].URLs[0] != "https://downloads.example/file.zip?token=private" {
		t.Fatal("mirror did not preserve pinned digest or signed URL")
	}
	for _, link := range []string{"a.com/file.zip", "http://example.com/file.zip", "file:///C:/file.zip", "https://user:password@example.com/file.zip", "https://example.com/file.zip#fragment", "https://example.com/file.7z", "https://example.com/file.rar"} {
		if _, err := CustomZIP(link, "", false); err == nil {
			t.Fatal("invalid custom link accepted", link)
		}
	}
	for _, digest := range []string{"bad", strings.Repeat("g", 64)} {
		if _, err := CustomZIP("https://example.com/file.zip", digest, true); err == nil {
			t.Fatal("invalid digest accepted")
		}
	}
	if _, err := CustomZIP("https://example.com/file.zip", strings.Repeat("a", 64), false); err == nil {
		t.Fatal("custom binary trust was not required")
	}
}

func TestCustomZIPRedirectExtractionAndSecretRedaction(t *testing.T) {
	data := zipBytes(t, map[string]string{"release/BIN/FFMPEG.EXE": "binary", "release/BIN/FFPROBE.EXE": "probe", "release/BIN/avcodec.dll": "sidecar", "release/LICENSE": "license"})
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "token=private" {
			t.Error("signed query was lost")
		}
		w.Write(data)
	}))
	defer target.Close()
	redirect := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/file?token=private", http.StatusFound)
	}))
	defer redirect.Close()
	certificates := x509.NewCertPool()
	certificates.AddCert(target.Certificate())
	certificates.AddCert(redirect.Certificate())
	installer := NewCustom(func(p Progress) {
		if strings.Contains(p.Source, "private") {
			t.Error("progress leaked URL query")
		}
	})
	transport := installer.Client.Transport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{RootCAs: certificates, MinVersion: tls.VersionTLS12}
	pkg, err := CustomZIP(redirect.URL+"/download?token=private", fmt.Sprintf("%x", sha256.Sum256(data)), true)
	if err != nil {
		t.Fatal(err)
	}
	result, err := installer.Install(context.Background(), t.TempDir(), []Package{pkg})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(result.FFmpeg) != "FFMPEG.EXE" || filepath.Base(result.FFprobe) != "FFPROBE.EXE" {
		t.Fatal("binary names were not detected")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(result.FFmpeg), "avcodec.dll")); err != nil {
		t.Fatal("sidecar lost", err)
	}
	notice, err := os.ReadFile(filepath.Join(result.Directory, "kompeg-download.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(notice), "private") {
		t.Fatal("notice leaked query string")
	}
}

func TestCustomDownloadFailuresLeaveNoInstallation(t *testing.T) {
	for _, test := range []struct {
		name    string
		content []byte
		message string
	}{
		{"page", []byte("<html>download page</html>"), "不是有效的 ZIP"},
		{"source", zipBytes(t, map[string]string{"src/ffmpeg.c": "source"}), "缺少 ffmpeg.exe"},
		{"missing probe", zipBytes(t, map[string]string{"bin/ffmpeg.exe": "binary"}), "缺少同目录 ffprobe.exe"},
		{"multiple", zipBytes(t, map[string]string{"a/ffmpeg.exe": "one", "b/ffmpeg.exe": "two"}), "多套 FFmpeg"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(test.content) }))
			defer server.Close()
			installer := NewCustom(nil)
			installer.Client.Transport = server.Client().Transport
			pkg, err := CustomZIP(server.URL+"/download.zip", fmt.Sprintf("%x", sha256.Sum256(test.content)), true)
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			_, err = installer.Install(context.Background(), root, []Package{pkg})
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("wrong error: %v", err)
			}
			entries, _ := os.ReadDir(root)
			if len(entries) != 0 {
				t.Fatal("failure published files")
			}
		})
	}
}

func TestCustomRedirectRejectsDowngradeAndCredentials(t *testing.T) {
	installer := NewCustom(nil)
	for _, link := range []string{"http://example.com/download.zip", "https://user:password@example.com/file.zip"} {
		req, _ := http.NewRequest(http.MethodGet, link, nil)
		if err := installer.Client.CheckRedirect(req, nil); err == nil {
			t.Fatal("unsafe redirect accepted", link)
		}
	}
	req, _ := http.NewRequest(http.MethodGet, "https://cdn.example/file.zip", nil)
	if err := installer.Client.CheckRedirect(req, nil); err != nil {
		t.Fatal("CDN redirect rejected", err)
	}
	if err := installer.Client.CheckRedirect(req, make([]*http.Request, 5)); err == nil {
		t.Fatal("redirect loop accepted")
	}
}
