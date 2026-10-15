package compiler

import (
	"kompeg/internal/core"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func subtitleSettings() core.SubtitleBurn {
	return core.SubtitleBurn{Source: "embedded", StreamIndex: 2, Font: "Microsoft YaHei", FontSize: 24, Color: "#ffffff", OutlineColor: "#000000", Outline: 2, Alignment: 2, Margin: 20}
}
func TestEditingRejectsInvalidRangesAndStyleInjection(t *testing.T) {
	for name, change := range map[string]func(*core.Settings){
		"too many":         func(s *core.Settings) { s.Segments = make([]core.TimeRange, 17) },
		"conflicting":      func(s *core.Settings) { s.Trim = &core.TimeRange{StartSeconds: 0, EndSeconds: 1} },
		"nonfinite":        func(s *core.Settings) { s.Segments[0].EndSeconds = math.Inf(1) },
		"operation":        func(s *core.Settings) { s.Operation = "remux" },
		"font expression":  func(s *core.Settings) { s.SubtitleBurn.Font = "Arial',movie=x" },
		"color expression": func(s *core.Settings) { s.SubtitleBurn.Color = "#fff:movie=x" },
		"bad size":         func(s *core.Settings) { s.SubtitleBurn.FontSize = 0 },
		"bad offset":       func(s *core.Settings) { s.SubtitleBurn.OffsetSeconds = math.NaN() },
	} {
		t.Run(name, func(t *testing.T) {
			s := core.DefaultSettings()
			s.Segments = []core.TimeRange{{StartSeconds: 1, EndSeconds: 2}}
			b := subtitleSettings()
			s.SubtitleBurn = &b
			change(&s)
			if err := ValidateSettings(s); err == nil {
				t.Fatal("invalid editing intent accepted")
			}
		})
	}
}
func TestSegmentsUseOrderedDurationBudgetAndEnforceFilters(t *testing.T) {
	p, m, c := fixture(t)
	p.Settings.Segments = []core.TimeRange{{StartSeconds: 6, EndSeconds: 8}, {StartSeconds: 1, EndSeconds: 2}}
	p.Settings.Mode = "size"
	p.Settings.TargetSizeMB = 2
	c.Filters = []string{"concat", "trim", "setpts", "atrim", "asetpts"}
	plan, err := Compile(p, m, c)
	if err != nil {
		t.Fatal(err)
	}
	if plan.DurationSeconds != 3 || !hasArguments(plan.Args, []string{"-ss", "6", "-t", "2"}) || !strings.Contains(strings.Join(plan.Args, " "), "concat=n=2:v=1:a=1") {
		t.Fatal("incorrect segment plan", plan)
	}
	if !contains(plan.Args, "5034k") {
		t.Fatal("size budget did not use selected duration", plan.Args)
	}
	p.Settings.Segments[0].StartSeconds = 7
	if strings.Contains(strings.Join(plan.Args, " "), "-ss 7") {
		t.Fatal("plan aliases editable ranges")
	}
	c.Filters = nil
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("missing filters accepted")
	}
}
func TestSubtitleAssetRequiresTextAndSupportedBuildAndNeverEmbedsPath(t *testing.T) {
	p, m, c := fixture(t)
	c.Filters = []string{"subtitles", "setpts"}
	c.Encoders = append(c.Encoders, core.Encoder{Name: "ass"})
	b := subtitleSettings()
	p.Settings.SubtitleBurn = &b
	m.Streams = append(m.Streams, core.Stream{Index: 2, CodecType: "subtitle", CodecName: "hdmv_pgs_subtitle"})
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("bitmap subtitle accepted")
	}
	path := filepath.Join(t.TempDir(), "字幕 ',[;].srt")
	if err := os.WriteFile(path, []byte("1\n00:00:00,000 --> 00:00:01,000\nx\n"), 0600); err != nil {
		t.Fatal(err)
	}
	p.Settings.SubtitleBurn.Source = "external"
	p.Settings.SubtitleBurn.Path = path
	plan, err := Compile(p, m, c)
	if err != nil {
		t.Fatal(err)
	}
	if plan.SubtitleAsset == nil || plan.SubtitleAsset.Fingerprint == "" || plan.SubtitleAsset.Format != "srt" {
		t.Fatal("asset identity missing")
	}
	if strings.Contains(strings.Join(plan.Args, " "), path) {
		t.Fatal("unescaped subtitle path entered filter")
	}
	c.Filters = nil
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("missing subtitle filter accepted")
	}
}
