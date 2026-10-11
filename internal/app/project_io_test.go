package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"kompeg/internal/core"
)

type projectProbeRunner struct{ count int }

func (r *projectProbeRunner) Probe(_ context.Context, input core.Source) (core.Media, error) {
	r.count++
	return core.Media{Source: input, DurationSeconds: 10, Streams: []core.Stream{{Index: 0, CodecType: "video", CodecName: "h264", Width: 160, Height: 90}, {Index: 1, CodecType: "audio", CodecName: "aac"}}}, nil
}
func (*projectProbeRunner) Capabilities(context.Context) (core.Capabilities, error) {
	return core.Capabilities{}, nil
}
func (*projectProbeRunner) Start(context.Context, core.ExecutionPlan, core.Callbacks) (core.Result, error) {
	return core.Result{}, errors.New("test probe adapter does not execute jobs")
}
func (*projectProbeRunner) Cancel(string) error { return nil }

func TestProjectIOReprobesPreservesIntentAndNeverExecutes(t *testing.T) {
	s, projects := batchService(t)
	p := projects[0]
	p.Settings.Trim = &core.TimeRange{StartSeconds: 1.5, EndSeconds: 4}
	p.Settings.AudioSelection = &core.TrackSelection{Mode: "selected", Indices: []int{1}}
	p.Settings.Transform = &core.VideoTransform{Rotation: 90, Crop: &core.CropMargins{Left: 2}}
	path := filepath.Join(t.TempDir(), "saved.ffproject")
	if err := saveProjectAt(path, p); err != nil {
		t.Fatal(err)
	}
	runner := &projectProbeRunner{}
	s.runner.replace(runner)
	loaded, err := s.loadProjectAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if runner.count != 1 || loaded.Project.Settings.Trim.StartSeconds != 1.5 || loaded.Project.Settings.Transform.Crop.Left != 2 || loaded.Project.Settings.AudioSelection.Indices[0] != 1 || loaded.Project.ID == p.ID || len(s.jobs.List()) != 0 {
		t.Fatal("project restoration changed intent or auto-enqueued")
	}
	if err := os.WriteFile(p.Output.URI, []byte("existing result"), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err = s.loadProjectAt(path)
	if err != nil || loaded.Project.Output.URI != "" {
		t.Fatal("existing output was not reset for a new choice", err)
	}
	if err := saveProjectAt(p.Inputs[0].URI, p); err == nil {
		t.Fatal("project overwrote source")
	}
	alias := filepath.Join(t.TempDir(), "alias.ffproject")
	if err := os.Link(p.Inputs[0].URI, alias); err == nil {
		if err := saveProjectAt(alias, p); err == nil {
			t.Fatal("hard-link media alias overwritten")
		}
	}
	if err := os.Remove(p.Inputs[0].URI); err != nil {
		t.Fatal(err)
	}
	if _, err := s.loadProjectAt(path); err == nil {
		t.Fatal("missing project media silently accepted")
	}
}

func TestPresetSharingStripsMediaDataAndDoesNotChangeDefault(t *testing.T) {
	s, projects := batchService(t)
	settings := projects[0].Settings
	settings.Trim = &core.TimeRange{StartSeconds: 2, EndSeconds: 5}
	settings.AudioSelection = &core.TrackSelection{Mode: "selected", Indices: []int{1}}
	settings.SubtitleSelection = &core.TrackSelection{Mode: "selected", Indices: []int{3}}
	settings.Transform = &core.VideoTransform{Rotation: 90, Crop: &core.CropMargins{Left: 2}}
	p := core.Preset{SchemaVersion: 1, ID: "shared", Name: "共享配置", Settings: settings}
	path := filepath.Join(t.TempDir(), "shared.pheebpreset")
	if err := exportPresetAt(path, p); err != nil {
		t.Fatal(err)
	}
	read, err := readPreset(path)
	if err != nil {
		t.Fatal(err)
	}
	if read.Settings.Trim != nil || read.Settings.AudioSelection != nil || read.Settings.SubtitleSelection != nil || read.Settings.Transform.Crop != nil || read.Settings.Transform.Rotation != 90 {
		t.Fatal("portable preset leaked media settings or lost general settings")
	}
	before, err := s.SaveSettings(projects[0].Settings, "")
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := s.importPresetAt(path)
	if err != nil || len(loaded.Presets) != 1 || loaded.LastSettings.Quality != before.LastSettings.Quality {
		t.Fatal("import altered defaults", err)
	}
	if _, err := s.importPresetAt(path); err == nil {
		t.Fatal("same-name preset silently replaced")
	}
	p.Settings.Quality = 20
	if err := exportPresetAt(path, p); err != nil {
		t.Fatal(err)
	}
	read, err = readPreset(path)
	if err != nil || read.Settings.Quality != 20 {
		t.Fatal("export did not replace the expressly selected old file", err)
	}
	bad := []byte(`{"schemaVersion":1,"name":"bad","settings":{"operation":"compress","codec":"h264","audio":"aac","mode":"quality","rawArguments":["-x264-params","pass=1"]}}`)
	if err := os.WriteFile(path, bad, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.importPresetAt(path); err == nil {
		t.Fatal("unsafe shared arguments accepted")
	}
	if len(s.GetWorkspace().Preferences.Presets) != 1 {
		t.Fatal("rejected import changed saved preferences")
	}
}

func TestPortablePresetPreservesGenericTrackPolicies(t *testing.T) {
	s, projects := batchService(t)
	settings := projects[0].Settings
	settings.AudioSelection = &core.TrackSelection{Mode: "none"}
	settings.SubtitleSelection = &core.TrackSelection{Mode: "all"}
	saved, err := s.SaveSettings(settings, "通用轨道策略")
	if err != nil {
		t.Fatal(err)
	}
	got := saved.Presets[0].Settings
	if got.AudioSelection == nil || got.AudioSelection.Mode != "none" || got.SubtitleSelection == nil || got.SubtitleSelection.Mode != "all" {
		t.Fatal("portable settings discarded generic track policies")
	}
}
