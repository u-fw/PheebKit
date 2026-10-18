package compiler

import (
	"math"
	"strings"
	"testing"

	"pheebkit/internal/core"
)

func audioSettings() core.Settings {
	s := core.DefaultSettings()
	s.AudioProcessing = &core.AudioProcessing{NoiseFloorDB: -50, Channels: "source"}
	return s
}
func TestAudioProcessingValidation(t *testing.T) {
	for name, mutate := range map[string]func(*core.Settings){
		"nan":         func(s *core.Settings) { s.AudioProcessing.FadeInSeconds = math.NaN() },
		"infinite":    func(s *core.Settings) { s.AudioProcessing.DenoiseReductionDB = math.Inf(1) },
		"noise floor": func(s *core.Settings) { s.AudioProcessing.DenoiseReductionDB = 12; s.AudioProcessing.NoiseFloorDB = -5 },
		"eq": func(s *core.Settings) {
			s.AudioProcessing.Equalizer = []core.EqualizerBand{{FrequencyHz: 1000, Q: 0, GainDB: 2}}
		},
		"nyquist": func(s *core.Settings) {
			s.AudioProcessing.SampleRate = 8000
			s.AudioProcessing.Equalizer = []core.EqualizerBand{{FrequencyHz: 5000, Q: 1, GainDB: 2}}
		},
		"loudness": func(s *core.Settings) {
			s.AudioProcessing.Normalize = &core.Loudness{IntegratedLUFS: 0, TruePeakDBTP: -1, RangeLU: 7}
		},
		"channels": func(s *core.Settings) { s.AudioProcessing.Channels = "stereo;movie=secret" },
		"rate":     func(s *core.Settings) { s.AudioProcessing.SampleRate = 12345 },
		"opus":     func(s *core.Settings) { s.Audio = "opus"; s.AudioProcessing.SampleRate = 16000 },
		"mp3": func(s *core.Settings) {
			s.Operation = "extract-audio"
			s.Audio = "mp3"
			s.AudioProcessing.SampleRate = 96000
		},
		"remux":    func(s *core.Settings) { s.Operation = "remux" },
		"copy":     func(s *core.Settings) { s.Audio = "copy" },
		"none":     func(s *core.Settings) { s.Audio = "none" },
		"recovery": func(s *core.Settings) { s.Recovery = &core.Recovery{} },
	} {
		t.Run(name, func(t *testing.T) {
			s := audioSettings()
			mutate(&s)
			if ValidateSettings(s) == nil {
				t.Fatal("invalid audio intent accepted")
			}
		})
	}
	s := audioSettings()
	s.AudioProcessing.FadeInSeconds = 2
	s.AudioProcessing.FadeOutSeconds = 3
	if ValidateAudioDuration(s, 4) == nil || ValidateAudioDuration(s, 0) == nil || ValidateAudioDuration(s, 5) != nil {
		t.Fatal("fade duration policy failed")
	}
	s.AudioProcessing.Equalizer = []core.EqualizerBand{{FrequencyHz: 8000, Q: 1, GainDB: 0}}
	s.AudioProcessing.SampleRate = 8000
	if ValidateSettings(s) != nil {
		t.Fatal("inactive EQ gain blocked rate selection")
	}
	copy := core.CloneSettings(s)
	copy.AudioProcessing.Equalizer[0].GainDB = 5
	if s.AudioProcessing.Equalizer[0].GainDB != 0 {
		t.Fatal("equalizer snapshot aliases source")
	}
	s.AudioProcessing.Normalize = &core.Loudness{IntegratedLUFS: -16, TruePeakDBTP: -1.5, RangeLU: 11}
	copy = core.CloneSettings(s)
	copy.AudioProcessing.Normalize.IntegratedLUFS = -20
	if s.AudioProcessing.Normalize.IntegratedLUFS != -16 {
		t.Fatal("loudness snapshot aliases source")
	}
}

func TestAudioProcessingRequiresRealFiltersAndRunsAfterConcat(t *testing.T) {
	p, m, c := trackFixture(t)
	p.Settings = audioSettings()
	p.Settings.Segments = []core.TimeRange{{EndSeconds: 2}, {StartSeconds: 3, EndSeconds: 5}}
	p.Settings.AudioProcessing.FadeInSeconds = 1
	p.Settings.AudioProcessing.FadeOutSeconds = 1
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("missing filters accepted")
	}
	c.Filters = []string{"aresample", "afade", "concat", "trim", "setpts", "atrim", "asetpts"}
	plan, err := Compile(p, m, c)
	if err != nil {
		t.Fatal(err)
	}
	graph := ""
	for i, arg := range plan.Args {
		if arg == "-filter_complex" {
			graph = plan.Args[i+1]
		}
	}
	if !strings.Contains(graph, "[joineda0]aresample=") || !strings.Contains(graph, "afade=t=out:st=3:d=1") || strings.Contains(strings.Join(plan.Args, " "), " -af ") {
		t.Fatal("audio effects not applied after concat", graph)
	}
	if plan.ExpectedOutput.AudioSampleRate != 48000 {
		t.Fatal("missing sample rate expectation")
	}
	p.Settings.AudioSelection = &core.TrackSelection{Mode: "none"}
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("effects silently skipped without selected audio")
	}
}
