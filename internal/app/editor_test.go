package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pheebkit/internal/core"
	"pheebkit/internal/localstore"
)

func TestRecoveryDraftRevisionsRestoreAndActivityProtection(t *testing.T) {
	s, projects := batchService(t)
	p := projects[0]
	p.Settings.Trim = &core.TimeRange{StartSeconds: 1, EndSeconds: 3}
	s.runner.replace(&projectProbeRunner{})
	if err := s.SaveRecoveryDraft(p, 10); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearRecoveryDraft(12); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveRecoveryDraft(p, 11); err != nil {
		t.Fatal(err)
	}
	data, path, err := readEditor()
	if err != nil || len(data.Drafts) != 0 {
		t.Fatal("stale save resurrected discarded draft", err)
	}
	if err := s.SaveRecoveryDraft(p, 13); err != nil {
		t.Fatal(err)
	}
	data, _, _ = readEditor()
	id := data.Drafts[0].ID
	if len(s.GetProjectLibrary().Drafts) != 0 {
		t.Fatal("own current checkpoint shown as recovery candidate")
	}
	if _, err := s.RestoreRecoveryDraft(id); err == nil {
		t.Fatal("restored live draft")
	}
	if err := s.RemoveRecoveryDraft(id); err == nil {
		t.Fatal("deleted live draft")
	}
	if err := localstore.Update(path, &data, func() error { data.Drafts[0].PID = 0; return nil }); err != nil {
		t.Fatal(err)
	}
	reopened := NewService()
	t.Cleanup(reopened.Close)
	reopened.runner.replace(&projectProbeRunner{})
	if len(reopened.GetProjectLibrary().Drafts) != 1 || reopened.GetProjectLibrary().Drafts[0].Active {
		t.Fatal("inactive previous-session draft was not offered")
	}
	restored, err := reopened.RestoreRecoveryDraft(id)
	if err != nil || !restored.NeedsSave || restored.Project.Settings.Trim.StartSeconds != 1 || len(s.jobs.List()) != 0 {
		t.Fatal("restore lost intent or executed work", err)
	}
	data, _, _ = readEditor()
	if len(data.Drafts) != 1 {
		t.Fatal("opening draft deleted the only recovery copy")
	}
	if err := os.Remove(p.Inputs[0].URI); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RestoreRecoveryDraft(id); err == nil {
		t.Fatal("missing media silently accepted")
	}
	data, _, _ = readEditor()
	if len(data.Drafts) != 1 {
		t.Fatal("failed recovery destroyed draft")
	}
	if err := s.RemoveRecoveryDraft(id); err != nil {
		t.Fatal(err)
	}
	data, _, _ = readEditor()
	if len(data.Drafts) != 0 {
		t.Fatal("explicit deletion did not persist")
	}
}

func TestEditorPreferencesBoundRetentionAndPreserveActiveDrafts(t *testing.T) {
	s, projects := batchService(t)
	if err := s.SaveRecoveryDraft(projects[0], 1); err != nil {
		t.Fatal(err)
	}
	data, path, _ := readEditor()
	now := time.Now()
	if err := localstore.Update(path, &data, func() error {
		data.Drafts = append(data.Drafts,
			RecoveryDraft{ID: "newest", UpdatedAt: now.Add(-time.Minute), Project: projects[0]},
			RecoveryDraft{ID: "older", UpdatedAt: now.Add(-time.Hour), Project: projects[0]},
			RecoveryDraft{ID: "expired", UpdatedAt: now.Add(-72 * time.Hour), Project: projects[0]},
			RecoveryDraft{ID: "other-active", PID: os.Getpid(), UpdatedAt: now.Add(-72 * time.Hour), Project: projects[0]})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	prefs := defaultEditorPreferences()
	prefs.DraftLimit, prefs.RetentionDays, prefs.Autosave = 1, 1, false
	if err := s.SaveEditorPreferences(prefs); err != nil {
		t.Fatal(err)
	}
	data, _, _ = readEditor()
	if len(data.Drafts) != 3 {
		t.Fatalf("retention did not bound inactive drafts or discarded active work: %+v", data.Drafts)
	}
	if err := s.SaveRecoveryDraft(projects[0], 2); err == nil {
		t.Fatal("autosave wrote despite disabled setting")
	}
	for _, mutate := range []func(*EditorPreferences){func(p *EditorPreferences) { p.IntervalSeconds = 0 }, func(p *EditorPreferences) { p.DraftLimit = 21 }, func(p *EditorPreferences) { p.RetentionDays = 91 }, func(p *EditorPreferences) { p.RecentLimit = -1 }, func(p *EditorPreferences) { p.UndoLimit = 201 }} {
		invalid := prefs
		mutate(&invalid)
		if err := s.SaveEditorPreferences(invalid); err == nil {
			t.Fatal("invalid editor preferences accepted")
		}
	}
}

func TestRecentProjectsReopenAndClearWithoutDeletingFiles(t *testing.T) {
	s, projects := batchService(t)
	s.runner.replace(&projectProbeRunner{})
	path := filepath.Join(t.TempDir(), "saved.ffproject")
	if err := saveProjectAt(path, projects[0]); err != nil {
		t.Fatal(err)
	}
	s.rememberProject(path)
	first := s.GetProjectLibrary()
	if first.Error != "" || len(first.Recent) != 1 {
		t.Fatal("recent record missing", first.Error)
	}
	s.rememberProject(path)
	if len(s.GetProjectLibrary().Recent) != 1 {
		t.Fatal("duplicate recent record")
	}
	value, err := s.OpenRecentProject(first.Recent[0].ID)
	if err != nil || value.Media.Source.URI != projects[0].Inputs[0].URI || len(s.jobs.List()) != 0 {
		t.Fatal("recent open did not re-probe safely", err)
	}
	if err := s.ClearRecentProjects(); err != nil {
		t.Fatal(err)
	}
	if len(s.GetProjectLibrary().Recent) != 0 {
		t.Fatal("recent list was not cleared")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("clearing recent records removed project file", err)
	}
	prefs := defaultEditorPreferences()
	prefs.RecentLimit = 0
	if err := s.SaveEditorPreferences(prefs); err != nil {
		t.Fatal(err)
	}
	s.rememberProject(path)
	if len(s.GetProjectLibrary().Recent) != 0 {
		t.Fatal("disabled recent tracking wrote paths")
	}
}

func TestCorruptEditorRecordIsNeverOverwritten(t *testing.T) {
	s, projects := batchService(t)
	path, err := workspaceFile("editor.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	original := []byte("{broken recovery data")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	if s.GetProjectLibrary().Error == "" {
		t.Fatal("corruption silently reset")
	}
	if err := s.SaveRecoveryDraft(projects[0], 1); err == nil {
		t.Fatal("corrupt record replaced by autosave")
	}
	if err := s.ClearRecoveryDraft(2); err == nil {
		t.Fatal("corrupt record replaced by clearing")
	}
	if err := s.SaveEditorPreferences(defaultEditorPreferences()); err == nil {
		t.Fatal("corrupt record replaced by preferences")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(original, after) {
		t.Fatal("corrupt original was changed")
	}
}

func TestOversizedDraftDoesNotReplaceLastCheckpoint(t *testing.T) {
	s, projects := batchService(t)
	p := projects[0]
	if err := s.SaveRecoveryDraft(p, 1); err != nil {
		t.Fatal(err)
	}
	_, path, _ := readEditor()
	before, _ := os.ReadFile(path)
	p.Inputs = []core.Source{{Kind: core.SourceLocalFile, URI: filepath.Join(t.TempDir(), strings.Repeat("x", 1024*1024))}}
	if err := s.SaveRecoveryDraft(p, 2); err == nil {
		t.Fatal("oversized recovery draft accepted")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("oversized save replaced valid checkpoint")
	}
}
