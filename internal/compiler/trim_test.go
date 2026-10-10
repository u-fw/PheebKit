package compiler

import (
	"math"
	"strings"
	"testing"

	"kompeg/internal/core"
)

func TestTrimRangeAndSizeBudget(t *testing.T) {
	p, m, c := fixture(t)
	p.Settings.Trim = &core.TimeRange{StartSeconds: 2.5, EndSeconds: 7.5}
	p.Settings.Mode, p.Settings.TargetSizeMB = "size", 5
	plan, err := Compile(p, m, c)
	if err != nil {
		t.Fatal(err)
	}
	if plan.DurationSeconds != 5 || !strings.Contains(strings.Join(plan.Args, "|"), strings.Join([]string{"-ss", "2.5", "-accurate_seek", "-i", p.Inputs[0].URI, "-t", "5", "-map_chapters", "-1"}, "|")) || !contains(plan.Args, "7648k") {
		t.Fatalf("wrong seek, duration, or trimmed size budget: %+v", plan)
	}
	p.Settings.Trim = nil
	plan, err = Compile(p, m, c)
	if err != nil || plan.DurationSeconds != 10 || contains(plan.Args, "-ss") || contains(plan.Args, "-t") || !contains(plan.Args, "3728k") {
		t.Fatalf("full-input behavior changed: %+v %v", plan, err)
	}
}

func TestRejectInvalidTrim(t *testing.T) {
	for _, trim := range []core.TimeRange{
		{StartSeconds: -1, EndSeconds: 5}, {StartSeconds: 5, EndSeconds: 5},
		{StartSeconds: 7, EndSeconds: 5}, {EndSeconds: 10.01},
		{StartSeconds: math.NaN(), EndSeconds: 5}, {EndSeconds: math.Inf(1)},
	} {
		p, m, c := fixture(t)
		p.Settings.Trim = &trim
		if _, err := Compile(p, m, c); err == nil {
			t.Fatalf("accepted invalid range %+v", trim)
		}
	}
	for _, duration := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		p, m, c := fixture(t)
		p.Settings.Trim = &core.TimeRange{EndSeconds: 1}
		m.DurationSeconds = duration
		if _, err := Compile(p, m, c); err == nil {
			t.Fatalf("accepted unknown duration %v", duration)
		}
	}
}

func TestTrimAudioBoundaryPolicy(t *testing.T) {
	p, m, c := fixture(t)
	p.Settings.Trim = &core.TimeRange{EndSeconds: 10}
	p.Settings.Audio = "copy"
	m.Streams[1].CodecName = "aac"
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("accepted packet-aligned copied audio for trim")
	}
	m.Streams = m.Streams[:1]
	if _, err := Compile(p, m, c); err != nil {
		t.Fatalf("silent video rejected: %v", err)
	}
}

func TestTrimRejectsSubframeRangesAndRawOverrides(t *testing.T) {
	p, m, c := fixture(t)
	m.Streams[0].FrameRate = 30
	p.Settings.Trim = &core.TimeRange{StartSeconds: 2, EndSeconds: 2.01}
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("accepted shorter-than-one-frame range")
	}
	p.Settings.Trim.EndSeconds = 3
	for _, option := range []string{"-ss", "-t", "-to", "-copyts"} {
		p.Settings.RawArguments = []string{option, "0"}
		if _, err := Compile(p, m, c); err == nil {
			t.Fatalf("raw option %s bypassed managed trim", option)
		}
	}
}
