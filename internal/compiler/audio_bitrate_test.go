package compiler

import "testing"

func TestCustomAudioBitrateChangesBudgetAndValidatesRange(t *testing.T) {
	p, media, caps := fixture(t)
	p.Settings.Mode, p.Settings.TargetSizeMB = "size", 1
	p.Settings.AudioBitrateKbps = 64
	plan, err := Compile(p, media, caps)
	if err != nil || !hasArguments(plan.Args, []string{"-b:a", "64k"}) || !hasArguments(plan.Args, []string{"-b:v", "720k"}) {
		t.Fatalf("custom audio budget not reflected: %v %v", plan.Args, err)
	}
	for _, bitrate := range []int{-1, 1, 321} {
		p.Settings.AudioBitrateKbps = bitrate
		if _, err := Compile(p, media, caps); err == nil {
			t.Fatal("invalid audio bitrate accepted")
		}
	}
	p.Settings.AudioBitrateKbps = 0
	plan, err = Compile(p, media, caps)
	if err != nil || !hasArguments(plan.Args, []string{"-b:a", "192k"}) {
		t.Fatal("legacy default audio bitrate changed", err)
	}
	p.Settings.Audio, p.Settings.Mode, p.Settings.AudioBitrateKbps = "none", "quality", 999
	if _, err := Compile(p, media, caps); err != nil {
		t.Fatal("inactive audio bitrate blocked a silent output", err)
	}
}
