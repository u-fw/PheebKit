package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"kompeg/internal/core"
)

type crashCheckpointMarker struct {
	PID     int    `json:"pid"`
	DraftID string `json:"draftID"`
}

// Executed only as an isolated child of TestEditorRecoveryAfterProcessKill.
// It runs the Go service without creating a Wails window or native UI.
func TestEditorCheckpointCrashHelper(t *testing.T) {
	if os.Getenv("PHEEBKIT_CRASH_HELPER") != "1" {
		t.Skip("subprocess fixture only")
	}
	root := os.Getenv("PHEEBKIT_CRASH_ROOT")
	if !filepath.IsAbs(root) {
		t.Fatal("fixture requires an absolute owned directory")
	}
	s := NewService()
	settings := core.DefaultSettings()
	settings.Codec, settings.Encoder = "h264", "cpu"
	settings.AudioBitrateKbps = 80
	settings.Trim = &core.TimeRange{StartSeconds: 1, EndSeconds: 3}
	p := core.Project{SchemaVersion: 1, ID: "crash-checkpoint", Inputs: []core.Source{source(filepath.Join(root, "source.mp4"))}, Output: source(filepath.Join(root, "result.mp4")), Settings: settings}
	if err := s.SaveRecoveryDraft(p, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveSettings(settings, "Before process termination"); err != nil {
		t.Fatal(err)
	}
	projectPath := filepath.Join(root, "saved.ffproject")
	if err := saveProjectAt(projectPath, p); err != nil {
		t.Fatal(err)
	}
	s.rememberProject(projectPath)
	data, _, err := readEditor()
	if err != nil || len(data.Drafts) != 1 {
		t.Fatal("checkpoint missing", err)
	}
	marker, _ := json.Marshal(crashCheckpointMarker{PID: os.Getpid(), DraftID: data.Drafts[0].ID})
	file, err := os.OpenFile(filepath.Join(root, "ready.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(marker); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	for {
		time.Sleep(time.Second)
	} // Parent kills this process without graceful cleanup.
}

func TestEditorRecoveryAfterProcessKill(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "source.mp4"), []byte("fixture identity; probe adapter supplied by parent"), 0600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestEditorCheckpointCrashHelper$", "-test.v")
	cmd.Env = append(os.Environ(), "PHEEBKIT_CRASH_HELPER=1", "PHEEBKIT_CRASH_ROOT="+root)
	hidePlatformProcess(cmd)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	var marker crashCheckpointMarker
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(filepath.Join(root, "ready.json"))
		if err == nil && json.Unmarshal(data, &marker) == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if marker.PID != cmd.Process.Pid || marker.DraftID == "" {
		t.Fatal("child did not commit a checkpoint")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	waited = true
	if err == nil {
		t.Fatal("child exited normally instead of being terminated")
	}
	s := NewService()
	defer s.Close()
	s.runner.replace(&projectProbeRunner{})
	library := s.GetProjectLibrary()
	if library.Error != "" || len(library.Drafts) != 1 || library.Drafts[0].Active || library.Drafts[0].ID != marker.DraftID {
		t.Fatalf("killed process checkpoint was not recoverable: %+v", library)
	}
	restored, err := s.RestoreRecoveryDraft(marker.DraftID)
	if err != nil || !restored.NeedsSave || restored.Project.Settings.AudioBitrateKbps != 80 || restored.Project.Settings.Trim.StartSeconds != 1 || len(s.jobs.List()) != 0 {
		t.Fatal("recovery lost committed intent or auto-executed", err)
	}
	workspace := s.GetWorkspace()
	if workspace.Error != "" || len(workspace.Preferences.Presets) != 1 || len(library.Recent) != 1 {
		t.Fatal("termination corrupted other workspace records", workspace.Error)
	}
}
