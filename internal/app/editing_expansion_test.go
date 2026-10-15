package app

import (
	"kompeg/internal/core"
	projectfile "kompeg/internal/project"
	"path/filepath"
	"testing"
)

func TestIndependentRangesBatchHasAtomicAdmissionAndKeepsSnapshots(t *testing.T) {
	s, projects := batchService(t)
	projects[1].Inputs = append([]core.Source(nil), projects[0].Inputs...)
	projects[0].Settings.Trim = &core.TimeRange{StartSeconds: 5, EndSeconds: 8}
	projects[1].Settings.Trim = &core.TimeRange{StartSeconds: 1, EndSeconds: 11}
	if _, err := s.StartBatch(projects); err == nil || len(s.jobs.List()) != 0 {
		t.Fatal("invalid range admitted a partial independent export")
	}
	projects[1].Settings.Trim.EndSeconds = 2
	jobs, err := s.StartBatch(projects)
	if err != nil || len(jobs) != 2 {
		t.Fatal(err)
	}
	projects[0].Settings.Trim.StartSeconds = 0
	stored, _ := s.jobs.Get(jobs[0].ID)
	if stored.Project.Settings.Trim.StartSeconds != 5 || stored.Progress.TotalTimeSeconds != 3 {
		t.Fatal("queue lost immutable selected range")
	}
}
func TestEditingRoundTripRecoveryAndPortablePresetExclusions(t *testing.T) {
	s, projects := batchService(t)
	p := projects[0]
	p.Settings.Segments = []core.TimeRange{{StartSeconds: 5, EndSeconds: 8}, {StartSeconds: 1, EndSeconds: 2}}
	p.Settings.SubtitleBurn = &core.SubtitleBurn{Source: "embedded", StreamIndex: 2, Font: "Microsoft YaHei", FontSize: 24, Color: "#ffffff", OutlineColor: "#000000", Outline: 2, Alignment: 2, Margin: 20}
	path := filepath.Join(t.TempDir(), "edits.ffproject")
	if err := saveProjectAt(path, p); err != nil {
		t.Fatal(err)
	}
	read, err := projectfile.Load(path)
	if err != nil || len(read.Settings.Segments) != 2 || read.Settings.SubtitleBurn.FontSize != 24 {
		t.Fatal("project round-trip lost edits", err)
	}
	if err := s.SaveRecoveryDraft(p, 1); err != nil {
		t.Fatal(err)
	}
	record, _, err := readEditor()
	if err != nil || len(record.Drafts) != 1 || len(record.Drafts[0].Project.Settings.Segments) != 2 || record.Drafts[0].Project.Settings.SubtitleBurn.StreamIndex != 2 {
		t.Fatal("checkpoint lost edits", err)
	}
	clone := core.CloneProject(p)
	clone.Settings.Segments[0].EndSeconds = 99
	clone.Settings.SubtitleBurn.Font = "Arial"
	if p.Settings.Segments[0].EndSeconds != 8 || p.Settings.SubtitleBurn.Font != "Microsoft YaHei" {
		t.Fatal("mutable clone aliases original")
	}
	portable := portableSettings(p.Settings)
	if len(portable.Segments) != 0 || portable.SubtitleBurn != nil {
		t.Fatal("media-specific edits escaped into portable presets")
	}
}
