package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pheebkit/internal/core"
	"pheebkit/internal/localstore"
)

func TestHistoryRecoveryAndMerge(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	data := historyFile{Version: 1, Entries: []historyEntry{
		{Job: core.Job{ID: "dead", State: core.JobRunning}, PID: 0},
		{Job: core.Job{ID: "alive", State: core.JobRunning}, PID: os.Getpid()},
		{Job: core.Job{ID: "done", State: core.JobCompleted, Project: &core.Project{Settings: core.DefaultSettings()}}, PID: 0},
	}}
	if err := localstore.Update(path, &data, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	recovered, err := loadHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered) != 2 || recovered[0].State != core.JobInterrupted || recovered[1].Project == nil {
		t.Fatalf("bad restoration: %+v", recovered)
	}
	if err := saveHistory(path, []core.Job{{ID: "new", State: core.JobCompleted}}); err != nil {
		t.Fatal(err)
	}
	if err := localstore.Read(path, &data); err != nil {
		t.Fatal(err)
	}
	if len(data.Entries) != 4 {
		t.Fatal("another instance's entries were overwritten")
	}
	for _, entry := range data.Entries {
		if entry.Job.ID == "alive" && entry.Job.State != core.JobRunning {
			t.Fatal("live instance marked interrupted")
		}
	}
	jobs := make([]core.Job, 210)
	for i := range jobs {
		jobs[i] = core.Job{ID: fmt.Sprintf("job-%d", i), State: core.JobCompleted, QueuedAt: time.Unix(int64(i+1), 0)}
	}
	if err := saveHistory(path, jobs); err != nil {
		t.Fatal(err)
	}
	if err := localstore.Read(path, &data); err != nil {
		t.Fatal(err)
	}
	if len(data.Entries) != 200 {
		t.Fatalf("history not bounded: %d", len(data.Entries))
	}
	found := false
	for _, entry := range data.Entries {
		if entry.Job.ID == "alive" {
			found = true
		}
	}
	if !found {
		t.Fatal("active task pruned")
	}
}

func TestPreferencesExcludeTrimAndUpdateNamedPreset(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	service := NewService()
	defer service.Close()
	settings := core.DefaultSettings()
	settings.Trim = &core.TimeRange{StartSeconds: 2, EndSeconds: 4}
	first, err := service.SaveSettings(settings, "分享")
	if err != nil {
		t.Fatal(err)
	}
	if first.LastSettings.Trim != nil || len(first.Presets) != 1 || first.Presets[0].Settings.Trim != nil {
		t.Fatal("media-specific trim persisted in preset")
	}
	settings.Quality = 22
	second, err := service.SaveSettings(settings, "分享")
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Presets) != 1 || second.Presets[0].Settings.Quality != 22 {
		t.Fatal("same-name update duplicated preset")
	}
	third, err := service.DeletePreset("分享")
	if err != nil || len(third.Presets) != 0 {
		t.Fatal("preset removal failed", err)
	}
	fresh := NewService()
	defer fresh.Close()
	if fresh.GetWorkspace().Preferences.LastSettings.Quality != 22 {
		t.Fatal("default lost on restart")
	}
	path, _ := workspaceFile("preferences.json")
	bad := []byte("{broken")
	os.WriteFile(path, bad, 0600)
	if fresh.GetWorkspace().Error == "" {
		t.Fatal("corruption not visible")
	}
	if _, err := fresh.SaveSettings(settings, ""); err == nil {
		t.Fatal("corrupt file overwritten")
	}
	actual, _ := os.ReadFile(path)
	if string(actual) != string(bad) {
		t.Fatal("original not preserved")
	}
}
