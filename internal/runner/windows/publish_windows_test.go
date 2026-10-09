//go:build windows

package windows

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtendedPathNormalizesWindowsSeparators(t *testing.T) {
	for input, expected := range map[string]string{
		`C:/Videos/子目录/../output.mp4`:   `\\?\C:\Videos\output.mp4`,
		`//server/share/子目录/output.mp4`: `\\?\UNC\server\share\子目录\output.mp4`,
		`\\?\C:\Videos\output.mp4`:      `\\?\C:\Videos\output.mp4`,
	} {
		if actual := extendedPath(input); actual != expected {
			t.Errorf("extendedPath(%q) = %q; want %q", input, actual, expected)
		}
	}
}

func TestPublicationAcceptsForwardSlashPaths(t *testing.T) {
	directory := t.TempDir()
	from := filepath.Join(directory, "临时 scratch.mp4")
	to := filepath.Join(directory, "输出 result.mp4")
	if err := os.WriteFile(from, []byte("encoded media"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := publishFile(filepath.ToSlash(from), filepath.ToSlash(to)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(to)
	if err != nil || string(data) != "encoded media" {
		t.Fatalf("incorrect published file: %q %v", data, err)
	}
	if _, err := os.Stat(from); !os.IsNotExist(err) {
		t.Fatalf("source was not moved: %v", err)
	}
}
