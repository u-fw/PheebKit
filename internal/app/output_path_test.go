package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOutputSuggestionProtectsExistingAndQueuedPaths(t *testing.T) {
	s, projects := batchService(t)
	path := projects[0].Output.URI
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(filepath.Dir(path), "output-0-2.mp4")
	if err := os.WriteFile(second, []byte("keep2"), 0600); err != nil {
		t.Fatal(err)
	}
	next, err := s.SuggestOutputPath(path)
	if err != nil || filepath.Base(next) != "output-0-3.mp4" {
		t.Fatal(next, err)
	}
	if bytes, _ := os.ReadFile(path); string(bytes) != "keep" {
		t.Fatal("suggestion overwrote output")
	}
	if _, err := s.StartBatch(projects[1:]); err != nil {
		t.Fatal(err)
	}
	next, err = s.SuggestOutputPath(projects[1].Output.URI)
	if err != nil || next == projects[1].Output.URI {
		t.Fatal("queued path reused", err)
	}
	if _, err := s.SuggestOutputPath("relative.mp4"); err == nil {
		t.Fatal("relative path accepted")
	}
}
func TestOutputDirectoryMemoryPreservesPreferencesAndHandlesMissingFolder(t *testing.T) {
	s, projects := batchService(t)
	if _, err := s.SaveSettings(projects[0].Settings, "保留"); err != nil {
		t.Fatal(err)
	}
	if err := s.RememberOutputDirectory(projects[0].Output.URI); err != nil {
		t.Fatal(err)
	}
	status := s.GetWorkspace()
	if len(status.Preferences.Presets) != 1 || status.Preferences.LastOutputDirectory != filepath.Dir(projects[0].Output.URI) {
		t.Fatal(status)
	}
	missing := filepath.Join(t.TempDir(), "missing", "out.mp4")
	if err := s.RememberOutputDirectory(missing); err == nil {
		t.Fatal("missing directory remembered")
	}
}
func TestUnsavedEditsProtectIdleCloseAndRejectStaleUpdates(t *testing.T) {
	s, _ := batchService(t)
	s.SetUnsavedEdits(true, 2)
	s.SetUnsavedEdits(false, 1)
	if s.CanClose() {
		t.Fatal("unsaved edit bypassed close protection")
	}
	s.SetUnsavedEdits(false, 3)
	if !s.CanClose() {
		t.Fatal("saved idle project cannot close")
	}
}
