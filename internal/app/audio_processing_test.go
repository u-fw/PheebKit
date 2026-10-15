package app

import (
	"path/filepath"
	"testing"

	"kompeg/internal/core"
	projectfile "kompeg/internal/project"
)

func TestAudioProcessingSnapshotsPersistenceAndSamplePolicy(t *testing.T) {
	s, projects := batchService(t)
	p := projects[0]
	m := s.media[p.Inputs[0].URI]
	m.Streams = append(m.Streams, core.Stream{Index: 1, CodecType: "audio", CodecName: "aac", SampleRate: 44100, Channels: 1})
	s.media[p.Inputs[0].URI] = m
	s.caps.Filters = []string{"aresample", "afade", "equalizer", "loudnorm"}
	p.Settings.AudioProcessing = &core.AudioProcessing{Channels: "stereo", NoiseFloorDB: -50, FadeInSeconds: 2, FadeOutSeconds: 2, Equalizer: []core.EqualizerBand{{FrequencyHz: 1000, Q: 1, GainDB: -3}}, Normalize: &core.Loudness{IntegratedLUFS: -16, TruePeakDBTP: -1.5, RangeLU: 11}}
	path := filepath.Join(t.TempDir(), "audio.ffproject")
	if err := saveProjectAt(path, p); err != nil {
		t.Fatal(err)
	}
	read, err := projectfile.Load(path)
	if err != nil || read.Settings.AudioProcessing.FadeOutSeconds != 2 {
		t.Fatal("project lost audio settings", err)
	}
	if err := s.SaveRecoveryDraft(p, 1); err != nil {
		t.Fatal(err)
	}
	record, _, err := readEditor()
	if err != nil || record.Drafts[0].Project.Settings.AudioProcessing.Normalize.IntegratedLUFS != -16 {
		t.Fatal("draft lost audio settings", err)
	}
	portable := portableSettings(p.Settings)
	portable.AudioProcessing.Equalizer[0].GainDB = 9
	if p.Settings.AudioProcessing.Equalizer[0].GainDB != -3 {
		t.Fatal("portable preset aliases source")
	}
	presetPath := filepath.Join(t.TempDir(), "audio.pheebpreset")
	if err := exportPresetAt(presetPath, core.Preset{SchemaVersion: 1, ID: "audio", Name: "audio", Settings: p.Settings}); err != nil {
		t.Fatal(err)
	}
	imported, err := s.importPresetAt(presetPath)
	if err != nil || len(imported.Presets) != 1 || imported.Presets[0].Settings.AudioProcessing.Channels != "stereo" {
		t.Fatal("portable preset lost audio processing", err)
	}
	job, err := s.StartSample(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	if job.Project.Settings.AudioProcessing.FadeInSeconds != 0 || job.Project.Settings.AudioProcessing.FadeOutSeconds != 0 || p.Settings.AudioProcessing.FadeInSeconds != 2 {
		t.Fatal("sample failed to isolate/disclose whole-output fades")
	}
	p.Settings.AudioProcessing.Normalize.IntegratedLUFS = -20
	stored, _ := s.jobs.Get(job.ID)
	if stored.Project.Settings.AudioProcessing.Normalize.IntegratedLUFS != -16 {
		t.Fatal("queued sample shares editable audio settings")
	}
	p.Settings.AudioProcessing.FadeInSeconds = 9
	p.Settings.AudioProcessing.FadeOutSeconds = 2
	if _, err := s.StartSample(p, 1); err == nil {
		t.Fatal("sample narrowing hid invalid full-output fades")
	}
}
