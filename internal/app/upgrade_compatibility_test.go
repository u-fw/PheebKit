package app

import (
	"bytes"
	"os"
	"path/filepath"
	"pheebkit/internal/compat"
	"testing"
)

func TestPreEditorWorkspaceLoadsWithoutRewritingLegacyFiles(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	directory := compat.LegacyConfigDirectory(os.Getenv("APPDATA"))
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"ffmpeg.json":      []byte(`{"ffmpeg":"","ffprobe":""}`),
		"preferences.json": []byte(`{"version":1,"presets":[],"lastSettings":{"codec":"hevc","quality":26,"preset":"medium","resolution":0,"audio":"aac","mode":"quality","bitrateKbps":0,"targetSizeMB":0,"rawArguments":[]}}`),
		"history.json":     []byte(`{"version":1,"entries":[]}`),
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := NewService()
	defer s.Close()
	workspace := s.GetWorkspace()
	if workspace.Error != "" || workspace.Preferences.LastSettings == nil || workspace.Preferences.LastSettings.Codec != "hevc" || workspace.Preferences.LastSettings.AudioBitrateKbps != 0 {
		t.Fatal("legacy settings did not retain defaults", workspace.Error)
	}
	library := s.GetProjectLibrary()
	if library.Error != "" || !library.Preferences.Autosave || library.Preferences.IntervalSeconds != 15 || len(s.jobs.List()) != 0 {
		t.Fatal("missing editor store was not backward compatible", library.Error)
	}
	for name, before := range files {
		after, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("legacy %s was rewritten on read: %v", name, err)
		}
	}
}
