package main

import (
	"kompeg/internal/app"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeDialogMissingRememberedDirectory(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "file.mp4")
	if err := os.WriteFile(filename, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"", filepath.Join(directory, "missing"), filename} {
		if got := dialogDirectory(path); got != "" {
			t.Fatalf("invalid default retained: %q", got)
		}
	}
	o := openOptions(app.FileDialog{Title: "选择媒体", Directory: directory, Filename: "测试 & clip.mp4", FilterName: "媒体", Pattern: "*.mp4;*.mkv"})
	if o.DefaultDirectory != directory || o.DefaultFilename != "测试 & clip.mp4" || len(o.Filters) != 1 || o.Filters[0].Pattern != "*.mp4;*.mkv" {
		t.Fatalf("dialog options lost intent: %+v", o)
	}
}
