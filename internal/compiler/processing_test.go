package compiler

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kompeg/internal/core"
)

func trackFixture(t *testing.T) (core.Project, core.Media, core.Capabilities) {
	p, m, c := fixture(t)
	m.Streams = []core.Stream{
		{Index: 0, CodecType: "video", CodecName: "h264", Width: 1920, Height: 1080, BitDepth: 8},
		{Index: 2, CodecType: "audio", CodecName: "flac", Language: "eng"},
		{Index: 4, CodecType: "audio", CodecName: "aac", Language: "zho"},
		{Index: 5, CodecType: "subtitle", CodecName: "subrip"},
		{Index: 7, CodecType: "subtitle", CodecName: "hdmv_pgs_subtitle"},
	}
	c.Encoders = append(c.Encoders, core.Encoder{Name: "libx264"}, core.Encoder{Name: "libopus"}, core.Encoder{Name: "subrip"}, core.Encoder{Name: "ass"}, core.Encoder{Name: "mov_text"}, core.Encoder{Name: "webvtt"}, core.Encoder{Name: "flac"}, core.Encoder{Name: "libmp3lame"}, core.Encoder{Name: "pcm_s16le"})
	return p, m, c
}

func selected(index ...int) *core.TrackSelection {
	return &core.TrackSelection{Mode: "selected", Indices: index}
}

func TestExplicitTracksUseGlobalIndexesAndSelectedAudioBudget(t *testing.T) {
	p, m, c := trackFixture(t)
	p.Settings.AudioSelection = selected(4)
	p.Settings.SubtitleSelection = selected(5)
	p.Settings.Mode, p.Settings.TargetSizeMB = "size", 5
	plan, err := Compile(p, m, c)
	if err != nil {
		t.Fatal(err)
	}
	if !hasArguments(plan.Args, []string{"-map", "0:0", "-map", "0:4", "-map", "0:5"}) || !contains(plan.Args, "3728k") || !hasArguments(plan.Args, []string{"-c:s:0", "mov_text"}) || contains(plan.Args, "-sn") {
		t.Fatalf("incorrect selected tracks/budget/subtitle conversion: %q", plan.Args)
	}
	p.Settings.Audio = "copy"
	p.Settings.Mode = "quality"
	if _, err := Compile(p, m, c); err != nil {
		t.Fatalf("unselected FLAC prevented selected AAC copy: %v", err)
	}
	p.Settings.AudioSelection = &core.TrackSelection{Mode: "none"}
	p.Settings.Mode = "size"
	plan, err = Compile(p, m, c)
	if err != nil || !contains(plan.Args, "3920k") || !contains(plan.Args, "-an") || contains(plan.Args, "0:4") {
		t.Fatalf("muted tracks affected audio budget: %q %v", plan.Args, err)
	}
}

func TestRejectInvalidExplicitSelections(t *testing.T) {
	for _, selection := range []*core.TrackSelection{selected(4, 4), selected(3), selected(5), selected(-1), selected(), {Mode: "first"}, {Mode: "all", Indices: []int{4}}} {
		p, m, c := trackFixture(t)
		p.Settings.AudioSelection = selection
		if _, err := Compile(p, m, c); err == nil {
			t.Fatalf("accepted invalid audio selection %+v", selection)
		}
	}
	p, m, c := trackFixture(t)
	p.Settings.AudioSelection = selected(4)
	m.Streams = append(m.Streams, m.Streams[2])
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("accepted ambiguous global stream index")
	}
}

func TestSubtitleContainerCompatibilityIsExplicit(t *testing.T) {
	for _, test := range []struct {
		ext       string
		index     int
		operation string
		encoder   string
		reject    bool
	}{
		{".mp4", 5, "", "mov_text", false}, {".mp4", 7, "", "", true},
		{".mkv", 7, "", "copy", false}, {".webm", 7, "", "", true},
		{".mp4", 5, "remux", "", true}, {".mkv", 5, "remux", "copy", false},
	} {
		t.Run(test.ext+test.operation+test.encoder, func(t *testing.T) {
			p, m, c := trackFixture(t)
			p.Output.URI = filepath.Join(t.TempDir(), "output"+test.ext)
			p.Settings.SubtitleSelection = selected(test.index)
			p.Settings.Operation = test.operation
			p.Settings.AudioSelection = &core.TrackSelection{Mode: "none"}
			plan, err := Compile(p, m, c)
			if test.reject {
				if err == nil {
					t.Fatal("accepted unsupported subtitle")
				}
				return
			}
			if err != nil || !hasArguments(plan.Args, []string{"-c:s:0", test.encoder}) {
				t.Fatalf("wrong subtitle handling %q %v", plan.Args, err)
			}
		})
	}
	p, m, c := trackFixture(t)
	p.Settings.SubtitleSelection = selected(5)
	c.Encoders = nil
	if _, err := compileSubtitles([]core.Stream{m.Streams[3]}, ".mp4", false, c.Encoders); err == nil {
		t.Fatal("accepted missing subtitle encoder")
	}
	p.Output.URI = filepath.Join(t.TempDir(), "copy.mkv")
	m.Streams[3].CodecName = "mov_text"
	p.Settings.SubtitleSelection = selected(5)
	plan, err := Compile(p, m, core.Capabilities{Encoders: []core.Encoder{{Name: "libx265"}, {Name: "aac"}, {Name: "subrip"}}})
	if err != nil || !hasArguments(plan.Args, []string{"-c:s:0", "subrip"}) {
		t.Fatalf("mov_text did not convert to supported Matroska codec: %q %v", plan.Args, err)
	}
}

func TestRemuxDoesNotNeedVideoOrAudioEncoders(t *testing.T) {
	p, m, _ := trackFixture(t)
	p.Output.URI = filepath.Join(t.TempDir(), "remux.mkv")
	p.Settings.Operation = "remux"
	p.Settings.SubtitleSelection = selected(5)
	p.Settings.AudioSelection = selected(4)
	m.Streams[0].HDR = "PQ / HDR10"
	plan, err := Compile(p, m, core.Capabilities{})
	if err != nil || plan.Encoder != "copy" || plan.DurationSeconds != 10 || contains(plan.Args, "-vf") || !hasArguments(plan.Args, []string{"-map_metadata", "0", "-map_chapters", "0"}) {
		t.Fatalf("remux performed encoding or lost duration/metadata: %+v %v", plan, err)
	}
	for _, mutate := range []func(*core.Settings){func(s *core.Settings) { s.Trim = &core.TimeRange{EndSeconds: 1} }, func(s *core.Settings) { s.VolumeDB = 1 }, func(s *core.Settings) { s.Transform = &core.VideoTransform{Rotation: 90} }, func(s *core.Settings) { s.Resolution = 720 }, func(s *core.Settings) { s.RawArguments = []string{"-metadata", "title=ignored"} }} {
		copy := core.CloneProject(p)
		mutate(&copy.Settings)
		if _, err := Compile(copy, m, core.Capabilities{}); err == nil {
			t.Fatal("remux silently ignored a requested change")
		}
	}
}

func TestAudioExtractionAllowsAudioOnlyAndChecksFormatAndCopyPolicy(t *testing.T) {
	for _, test := range []struct{ mode, ext, encoder string }{{"copy", ".mka", "copy"}, {"copy", ".m4a", "copy"}, {"aac", ".m4a", "aac"}, {"opus", ".opus", "libopus"}, {"mp3", ".mp3", "libmp3lame"}, {"flac", ".flac", "flac"}, {"wav", ".wav", "pcm_s16le"}} {
		t.Run(test.mode+test.ext, func(t *testing.T) {
			p, m, c := trackFixture(t)
			m.Streams = m.Streams[1:3]
			p.Settings.Operation = "extract-audio"
			p.Settings.AudioSelection = selected(4)
			p.Settings.Audio = test.mode
			p.Output.URI = filepath.Join(t.TempDir(), "audio"+test.ext)
			plan, err := Compile(p, m, c)
			if err != nil || !hasArguments(plan.Args, []string{"-map", "0:4"}) || !hasArguments(plan.Args, []string{"-c:a", test.encoder}) || !contains(plan.Args, "-vn") {
				t.Fatalf("bad extraction %q %v", plan.Args, err)
			}
			if test.mode != "copy" {
				p.Settings.Trim = &core.TimeRange{StartSeconds: 1, EndSeconds: 4}
				p.Settings.VolumeDB = -6
				plan, err = Compile(p, m, c)
				if err != nil || plan.DurationSeconds != 3 || !hasArguments(plan.Args, []string{"-af", "volume=-6dB"}) {
					t.Fatalf("encoded audio trim/volume failed: %+v %v", plan, err)
				}
			}
		})
	}
	p, m, c := trackFixture(t)
	p.Settings.Operation = "extract-audio"
	p.Settings.AudioSelection = selected(2)
	p.Settings.Audio = "copy"
	p.Output.URI = filepath.Join(t.TempDir(), "bad.m4a")
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("FLAC copied into M4A")
	}
	p.Output.URI = filepath.Join(t.TempDir(), "copy.mka")
	p.Settings.Trim = &core.TimeRange{EndSeconds: 3}
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("copy audio silently used packet-aligned trim")
	}
	p.Settings.Trim = nil
	p.Settings.AudioSelection = nil
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("audio extraction did not require deliberate one-track selection")
	}
	p.Settings.AudioSelection = selected(2, 4)
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("audio extraction accepted multiple tracks")
	}
}

func TestTextSubtitleExtractionRejectsBitmapAndUnknownEncoder(t *testing.T) {
	for _, ext := range []string{".srt", ".ass", ".vtt"} {
		p, m, c := trackFixture(t)
		p.Settings.Operation = "extract-subtitle"
		p.Settings.SubtitleSelection = selected(5)
		p.Output.URI = filepath.Join(t.TempDir(), "subtitle"+ext)
		plan, err := Compile(p, m, c)
		if err != nil || !contains(plan.Args, "-vn") || !contains(plan.Args, "-an") {
			t.Fatalf("text extraction %s %q %v", ext, plan.Args, err)
		}
	}
	p, m, c := trackFixture(t)
	p.Settings.Operation = "extract-subtitle"
	p.Settings.SubtitleSelection = selected(7)
	p.Output.URI = filepath.Join(t.TempDir(), "bitmap.srt")
	if _, err := Compile(p, m, c); err == nil || !strings.Contains(err.Error(), "OCR") {
		t.Fatalf("bitmap subtitle accepted or unclear: %v", err)
	}
	p.Settings.SubtitleSelection = selected(5)
	c.Encoders = nil
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("unavailable subtitle encoder accepted")
	}
}

func TestTransformsAreOrderedAndValidated(t *testing.T) {
	p, m, c := trackFixture(t)
	p.Settings.Transform = &core.VideoTransform{Crop: &core.CropMargins{Left: 100, Right: 100, Top: 20, Bottom: 20}, Rotation: 90, FlipHorizontal: true, FlipVertical: true}
	p.Settings.VolumeDB = -3
	plan, err := Compile(p, m, c)
	filter := "crop=w=1720:h=1040:x=100:y=20:exact=1,transpose=clock,hflip,vflip,scale=iw:ih,format=yuv420p"
	if err != nil || !hasArguments(plan.Args, []string{"-vf", filter}) || contains(plan.Args, "-noautorotate") || !hasArguments(plan.Args, []string{"-af", "volume=-3dB"}) {
		t.Fatalf("incorrect transform order %q %v", plan.Args, err)
	}
	for _, transform := range []*core.VideoTransform{{Rotation: 45}, {Crop: &core.CropMargins{Left: 1}}, {Crop: &core.CropMargins{Top: -2}}, {Crop: &core.CropMargins{Left: 1920}}, {Crop: &core.CropMargins{Bottom: 20000}}} {
		copy := core.CloneProject(p)
		copy.Settings.Transform = transform
		if _, err := Compile(copy, m, c); err == nil {
			t.Fatalf("accepted invalid transform %+v", transform)
		}
	}
	for _, volume := range []float64{math.NaN(), math.Inf(1), -61, 21} {
		copy := core.CloneProject(p)
		copy.Settings.VolumeDB = volume
		if _, err := Compile(copy, m, c); err == nil {
			t.Fatalf("accepted invalid volume %v", volume)
		}
	}
	p.Settings.Audio = "copy"
	p.Settings.AudioSelection = selected(4)
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("volume filter allowed audio stream copy")
	}
	p.Settings.Audio = "none"
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("volume filter silently ignored while muted")
	}
}

func TestClonedSettingsCannotMutateAnotherProjectSnapshot(t *testing.T) {
	p, _, _ := trackFixture(t)
	p.Settings.AudioSelection = selected(4)
	p.Settings.SubtitleSelection = selected(5)
	p.Settings.Transform = &core.VideoTransform{Crop: &core.CropMargins{Left: 4}}
	p.Settings.RawArguments = []string{"-metadata", "title=original"}
	p.Settings.Trim = &core.TimeRange{EndSeconds: 5}
	copy := core.CloneProject(p)
	copy.Inputs[0].URI = "changed"
	copy.Settings.AudioSelection.Indices[0] = 2
	copy.Settings.SubtitleSelection.Indices[0] = 7
	copy.Settings.Transform.Crop.Left = 100
	copy.Settings.RawArguments[1] = "changed"
	copy.Settings.Trim.EndSeconds = 9
	if p.Settings.AudioSelection.Indices[0] != 4 || p.Settings.SubtitleSelection.Indices[0] != 5 || p.Settings.Transform.Crop.Left != 4 || p.Settings.RawArguments[1] != "title=original" || p.Settings.Trim.EndSeconds != 5 || p.Inputs[0].URI == "changed" {
		t.Fatal("editable settings leaked into queue/preset snapshot")
	}
}

func TestSavedSettingsRejectUnsafeSharedArguments(t *testing.T) {
	for _, args := range [][]string{{"-y", "1"}, {"-x265-params", "analysis-save=secret"}, {"-x264-params", "pass=1"}, {"-x264-params", "aq-mode=1"}} {
		s := core.DefaultSettings()
		s.RawArguments = args
		if err := ValidateSettings(s); err == nil {
			t.Fatalf("shared settings accepted unsafe/incompatible arguments %q", args)
		}
	}
}

func TestDisplayOrientedCropUsesPreviewAxes(t *testing.T) {
	p, m, c := trackFixture(t)
	m.Streams[0].DisplayRotation = 270
	p.Settings.Transform = &core.VideoTransform{Crop: &core.CropMargins{Left: 100, Right: 100, Top: 20, Bottom: 20}}
	plan, err := Compile(p, m, c)
	if err != nil || !hasArguments(plan.Args, []string{"-vf", "crop=w=880:h=1880:x=100:y=20:exact=1,scale=iw:ih,format=yuv420p"}) {
		t.Fatalf("crop used raw rather than preview axes: %q %v", plan.Args, err)
	}
	m.Streams[0].DisplayRotation = 45
	if _, err := Compile(p, m, c); err == nil || !strings.Contains(err.Error(), "display rotation") {
		t.Fatalf("unsupported orientation cropped with wrong bounds: %v", err)
	}
}

func TestWebMCopyCompatibilityIsStrict(t *testing.T) {
	p, m, c := trackFixture(t)
	p.Settings.Operation = "remux"
	p.Output.URI = filepath.Join(t.TempDir(), "output.webm")
	p.Settings.AudioSelection = selected(4)
	p.Settings.SubtitleSelection = selected(5)
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("H264 accepted in WebM")
	}
	m.Streams[0].CodecName = "vp9"
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("AAC accepted in WebM")
	}
	m.Streams[2].CodecName = "opus"
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("SubRip copied into WebM")
	}
	m.Streams[3].CodecName = "webvtt"
	if _, err := Compile(p, m, core.Capabilities{}); err != nil {
		t.Fatalf("native WebM copy unnecessarily required encoder: %v", err)
	}
}

func TestOutputDirectoryMustExistBeforeQueueing(t *testing.T) {
	p, m, c := trackFixture(t)
	p.Output.URI = filepath.Join(t.TempDir(), "missing-directory", "output.mp4")
	if _, err := Compile(p, m, c); err == nil || !strings.Contains(err.Error(), "output directory") {
		t.Fatalf("nonexistent output folder accepted: %v", err)
	}
	parentFile := filepath.Join(t.TempDir(), "a-file")
	if err := os.WriteFile(parentFile, []byte("user-owned"), 0600); err != nil {
		t.Fatal(err)
	}
	p.Output.URI = filepath.Join(parentFile, "output.mp4")
	if _, err := Compile(p, m, c); err == nil {
		t.Fatal("file accepted as output folder")
	}
	if data, err := os.ReadFile(parentFile); err != nil || string(data) != "user-owned" {
		t.Fatalf("preflight modified user data: %s %v", data, err)
	}
}
