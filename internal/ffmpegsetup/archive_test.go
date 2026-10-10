package ffmpegsetup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ulikunitz/xz"
)

func tarBytes(t *testing.T, header *tar.Header) []byte {
	t.Helper()
	var b bytes.Buffer
	w := tar.NewWriter(&b)
	if header != nil {
		if err := w.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
	} else {
		for name, value := range map[string]string{"release/bin/ffmpeg.exe": "binary", "release/bin/ffprobe.exe": "probe", "release/bin/avcodec.dll": "sidecar", "release/LICENSE": "license"} {
			if err := w.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(value))}); err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte(value)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestCustomArchiveFormats(t *testing.T) {
	tarb := tarBytes(t, nil)
	var gz, xzb bytes.Buffer
	g := gzip.NewWriter(&gz)
	if _, err := g.Write(tarb); err != nil {
		t.Fatal(err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	x, err := xz.NewWriter(&xzb)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := x.Write(tarb); err != nil {
		t.Fatal(err)
	}
	if err := x.Close(); err != nil {
		t.Fatal(err)
	}
	formats := map[string][]byte{"tar": tarb, "tar.gz": gz.Bytes(), "tgz": gz.Bytes(), "tar.xz": xzb.Bytes()}
	for _, name := range []string{"tools.7z", "tools.rar", "tools.tar.bz2"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		formats[name] = data
	}
	for format, data := range formats {
		t.Run(format, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
			defer server.Close()
			installer := NewCustom(nil)
			installer.Client.Transport = server.Client().Transport
			// An extensionless link must work: inspect content, never URL suffix.
			pkg, err := CustomArchive(server.URL+"/download", fmt.Sprintf("%x", sha256.Sum256(data)), true)
			if err != nil {
				t.Fatal(err)
			}
			result, err := installer.Install(context.Background(), t.TempDir(), []Package{pkg})
			if err != nil {
				t.Fatal(err)
			}
			for name, want := range map[string]string{"bin/ffmpeg.exe": "binary", "bin/ffprobe.exe": "probe", "bin/avcodec.dll": "sidecar", "LICENSE": "license"} {
				got, err := os.ReadFile(filepath.Join(result.Directory, "release", filepath.FromSlash(name)))
				if err != nil || string(got) != want {
					t.Fatalf("lost payload %s: %s %v", name, got, err)
				}
			}
		})
	}
}

func TestTarRejectsUnsafeEntries(t *testing.T) {
	for _, h := range []*tar.Header{
		{Name: "../escape", Typeflag: tar.TypeReg},
		{Name: "C:/escape", Typeflag: tar.TypeReg},
		{Name: "bin/link", Typeflag: tar.TypeSymlink, Linkname: "../../outside"},
		{Name: "bin/hard", Typeflag: tar.TypeLink, Linkname: "../../outside"},
		{Name: "bin/device", Typeflag: tar.TypeChar},
	} {
		t.Run(h.Name, func(t *testing.T) {
			root := t.TempDir()
			archive := filepath.Join(root, "input")
			if err := os.WriteFile(archive, tarBytes(t, h), 0600); err != nil {
				t.Fatal(err)
			}
			if err := extract(context.Background(), archive, root, Artifact{Format: "auto"}); err == nil {
				t.Fatal("unsafe TAR accepted")
			}
		})
	}
}

func TestArchiveOutputLimitsAndCancellation(t *testing.T) {
	out := archiveOutput{ctx: context.Background(), directory: t.TempDir(), total: maxExpanded - 1}
	if err := out.write("oversized", 0600, 2, strings.NewReader("ab")); err == nil {
		t.Fatal("aggregate size limit ignored")
	}
	out.total = 0
	out.count = 10000
	if err := out.write("extra", 0600, 0, strings.NewReader("")); err == nil {
		t.Fatal("entry count ignored")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out.ctx = ctx
	if err := out.write("cancelled", 0600, 0, strings.NewReader("")); err != context.Canceled {
		t.Fatalf("cancellation ignored: %v", err)
	}
}

func TestDefaultSourceNeverDowngrades(t *testing.T) {
	for _, source := range []string{"stable", "auto"} {
		packages, err := Catalog(source)
		if err != nil || len(packages) != 1 || packages[0].ID != "gyan-9.0.2" {
			t.Fatalf("default source downgraded: %+v %v", packages, err)
		}
		if !strings.HasPrefix(packages[0].Artifacts[0].URLs[0], "https://www.gyan.dev/") {
			t.Fatal("Gyan must be tried first")
		}
	}
}

func TestSourcesAreTriedInDeclaredOrderWithoutProbes(t *testing.T) {
	data := zipBytes(t, map[string]string{"bin/ffmpeg.exe": "binary", "bin/ffprobe.exe": "probe"})
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		if r.Header.Get("Range") != "" {
			t.Error("unexpected speed probe")
		}
		if r.URL.Path == "/primary" {
			http.Error(w, "unavailable", 503)
			return
		}
		w.Write(data)
	}))
	defer server.Close()
	installer := &Installer{Client: server.Client()}
	pkg := Package{ID: "ordered", Artifacts: []Artifact{{Name: "ffmpeg.zip", Format: "zip", SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), URLs: []string{server.URL + "/primary", server.URL + "/backup", server.URL + "/unused"}}}}
	if _, err := installer.Install(context.Background(), t.TempDir(), []Package{pkg}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(requests, ",") != "/primary,/backup" {
		t.Fatalf("source order changed or extra source contacted: %v", requests)
	}
}
