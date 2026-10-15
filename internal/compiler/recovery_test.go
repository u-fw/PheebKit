package compiler

import (
	"math"
	"testing"

	"kompeg/internal/core"
)

func recoverySettings() core.Settings {
	s := core.DefaultSettings()
	s.Recovery = &core.Recovery{Mode: "delete", SourceFingerprint: "source", DamagedRanges: []core.TimeRange{{StartSeconds: 5, EndSeconds: 10}}}
	return s
}

func TestRecoveryTimelineAndValidation(t *testing.T) {
	s := recoverySettings()
	for _, mode := range []string{"delete", "black"} {
		s.Recovery.Mode = mode
		if err := ValidateSettings(s); err != nil {
			t.Fatal(err)
		}
		d, err := SelectedDuration(s, 16, 10)
		want := 11.0
		if mode == "black" {
			want = 16
		}
		if err != nil || d != want {
			t.Fatal(mode, d, err)
		}
	}
	for name, mutate := range map[string]func(*core.Settings){
		"mode":      func(s *core.Settings) { s.Recovery.Mode = "shell" },
		"copy":      func(s *core.Settings) { s.Audio = "copy" },
		"operation": func(s *core.Settings) { s.Operation = "remux" },
		"trim":      func(s *core.Settings) { s.Trim = &core.TimeRange{EndSeconds: 2} },
		"segments":  func(s *core.Settings) { s.Segments = []core.TimeRange{{EndSeconds: 2}} },
		"raw":       func(s *core.Settings) { s.RawArguments = []string{"-threads", "1"} },
		"empty":     func(s *core.Settings) { s.Recovery.DamagedRanges = nil },
		"nan":       func(s *core.Settings) { s.Recovery.DamagedRanges[0].StartSeconds = math.NaN() },
		"infinity":  func(s *core.Settings) { s.Recovery.DamagedRanges[0].EndSeconds = math.Inf(1) },
		"overlap": func(s *core.Settings) {
			s.Recovery.DamagedRanges = append(s.Recovery.DamagedRanges, core.TimeRange{StartSeconds: 9, EndSeconds: 11})
		},
		"adjacent": func(s *core.Settings) {
			s.Recovery.DamagedRanges = append(s.Recovery.DamagedRanges, core.TimeRange{StartSeconds: 10, EndSeconds: 11})
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := recoverySettings()
			mutate(&s)
			if ValidateSettings(s) == nil {
				t.Fatal("unsafe intent accepted")
			}
		})
	}
	for name, ranges := range map[string][]core.TimeRange{
		"all": {{EndSeconds: 16}}, "beyond": {{StartSeconds: 5, EndSeconds: 17}}, "tiny": {{StartSeconds: .01, EndSeconds: 5}},
	} {
		t.Run(name, func(t *testing.T) {
			s := recoverySettings()
			s.Recovery.DamagedRanges = ranges
			if _, err := SelectedDuration(s, 16, 10); err == nil {
				t.Fatal("invalid timeline accepted")
			}
		})
	}
	s = recoverySettings()
	s.Recovery.Mode = "black"
	s.Recovery.DamagedRanges = nil
	for i := 0; i < 9; i++ {
		s.Recovery.DamagedRanges = append(s.Recovery.DamagedRanges, core.TimeRange{StartSeconds: float64(i*2 + 1), EndSeconds: float64(i*2 + 2)})
	}
	if _, err := SelectedDuration(s, 20, 10); err == nil {
		t.Fatal("fragment limit ignored")
	}
	copy := core.CloneSettings(s)
	copy.Recovery.DamagedRanges[0].StartSeconds = 99
	if s.Recovery.DamagedRanges[0].StartSeconds == 99 {
		t.Fatal("snapshot aliases editable recovery")
	}
}
