package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kompeg/internal/core"
)

func fixture(t *testing.T) (core.Project, core.Media, core.Capabilities) {
	t.Helper()
	dir := t.TempDir()
	input := core.Source{URI: filepath.Join(dir, "输入 ' $ & ;.mkv"), Kind: core.SourceLocalFile}
	output := core.Source{URI: filepath.Join(dir, "output 中文.mp4"), Kind: core.SourceLocalFile}
	project := core.Project{SchemaVersion: 1, ID: "project", Inputs: []core.Source{input}, Output: output, Settings: core.DefaultSettings()}
	media := core.Media{Source: input, SizeBytes: 1000000, DurationSeconds: 10,
		Streams: []core.Stream{{CodecType: "video", CodecName: "h264", Width: 1920, Height: 1080, BitDepth: 8}, {CodecType: "audio", CodecName: "flac"}}}
	caps := core.Capabilities{Encoders: []core.Encoder{{Name: "libx265"}, {Name: "aac"}}}
	return project, media, caps
}

func TestCompileKeepsPathsAsIndividualArguments(t *testing.T) {
	project, media, caps := fixture(t)
	plan, err := Compile(project, media, caps)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Args[len(plan.Args)-1] != project.Output.URI || plan.Args[7] != project.Inputs[0].URI {
		t.Fatalf("paths were altered/split: %q", plan.Args)
	}
	if !strings.Contains(plan.Preview, "输入 '' $ & ;.mkv") || !strings.HasPrefix(plan.Preview, "& '") {
		t.Fatalf("PowerShell preview did not quote literally: %s", plan.Preview)
	}
	for _, required := range []string{"-n", "-progress", "pipe:1", "libx265", "-sn"} {
		if !contains(plan.Args, required) {
			t.Fatalf("missing %s: %q", required, plan.Args)
		}
	}
}

func TestRejectsUnsafeOrUnexecutableIntent(t *testing.T) {
	cases := map[string]func(*core.Project, *core.Media, *core.Capabilities){
		"same input output": func(p *core.Project, _ *core.Media, _ *core.Capabilities) { p.Output = p.Inputs[0] },
		"content URI": func(p *core.Project, _ *core.Media, _ *core.Capabilities) {
			p.Output.Kind = core.SourceContentURI
			p.Output.URI = "content://documents/42"
		},
		"unavailable encoder": func(_ *core.Project, _ *core.Media, c *core.Capabilities) {
			c.Encoders = []core.Encoder{{Name: "hevc_nvenc"}, {Name: "aac"}}
		},
		"overwrite raw": func(p *core.Project, _ *core.Media, _ *core.Capabilities) {
			p.Settings.RawArguments = []string{"-y", "1"}
		},
		"unsupported audio copy": func(p *core.Project, _ *core.Media, _ *core.Capabilities) { p.Settings.Audio = "copy" },
		"unprobed input":         func(_ *core.Project, m *core.Media, _ *core.Capabilities) { m.Source.URI += ".other" },
		"odd resolution":         func(p *core.Project, _ *core.Media, _ *core.Capabilities) { p.Settings.Resolution = 719 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			project, media, caps := fixture(t)
			mutate(&project, &media, &caps)
			if _, err := Compile(project, media, caps); err == nil {
				t.Fatal("accepted invalid intent")
			}
		})
	}
	project, media, caps := fixture(t)
	if err := os.WriteFile(project.Output.URI, []byte("existing output"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(project, media, caps); err == nil {
		t.Fatal("accepted existing output")
	}
}

func TestSizeBudgetRequiresKnownAudioRate(t *testing.T) {
	project, media, caps := fixture(t)
	project.Settings.Mode, project.Settings.TargetSizeMB = "size", 5
	plan, err := Compile(project, media, caps)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(plan.Args, "3728k") { // 5 MB * 8 * .98 / 10 s - 192 Kbps.
		t.Fatalf("incorrect video budget: %q", plan.Args)
	}
	project.Output.URI = filepath.Join(t.TempDir(), "copy.mkv")
	project.Settings.Audio = "copy"
	if _, err := Compile(project, media, caps); err == nil {
		t.Fatal("size mode accepted unknown copied audio budget")
	}
}

func TestCompressionSelectsMotionStreamAndRejectsOnlyCoverArt(t *testing.T) {
	project, media, caps := fixture(t)
	media.Streams = []core.Stream{
		{Index: 0, CodecType: "video", CodecName: "mjpeg", AttachedPicture: true},
		{Index: 1, CodecType: "video", CodecName: "mjpeg", TimedThumbnails: true},
		{Index: 3, CodecType: "video", CodecName: "h264", Width: 1920, Height: 1080},
	}
	plan, err := Compile(project, media, caps)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Args[9] != "0:3" {
		t.Fatalf("mapped cover art instead of motion video: %q", plan.Args)
	}
	media.Streams = media.Streams[:2]
	if _, err := Compile(project, media, caps); err == nil {
		t.Fatal("cover-art-only media accepted as a movie")
	}
}

func TestExplicitResolutionChoiceAlsoUpscales(t *testing.T) {
	project, media, caps := fixture(t)
	project.Settings.Resolution = 2160
	plan, err := Compile(project, media, caps)
	if err != nil {
		t.Fatal(err)
	}
	if !hasArguments(plan.Args, []string{"-vf", "scale=-2:2160,format=yuv420p", "-pix_fmt", "+yuv420p"}) {
		t.Fatalf("explicit output resolution was silently ignored: %q", plan.Args)
	}
	project.Settings.Resolution = 0
	plan, err = Compile(project, media, caps)
	if err != nil || !hasArguments(plan.Args, []string{"-vf", "scale=iw:ih,format=yuv420p", "-pix_fmt", "+yuv420p"}) {
		t.Fatalf("original resolution was changed: %q %v", plan.Args, err)
	}
}

func TestCPUFallbackRequiresExactTenBitOutput(t *testing.T) {
	project, media, caps := fixture(t)
	media.Streams[0].BitDepth = 10
	plan, err := Compile(project, media, caps)
	if err != nil {
		t.Fatal(err)
	}
	if plan.EncoderKind != "cpu" || !hasArguments(plan.Args, []string{"-vf", "scale=iw:ih,format=yuv420p10le", "-pix_fmt", "+yuv420p10le"}) {
		t.Fatalf("CPU fallback allowed implicit bit-depth conversion: %+v", plan)
	}
}

func TestHDRCompressionRequiresVerifiedPolicy(t *testing.T) {
	for _, hdr := range []string{"PQ / HDR10", "HLG", "Dolby Vision"} {
		t.Run(hdr, func(t *testing.T) {
			project, media, caps := fixture(t)
			media.Streams[0].HDR, media.Streams[0].BitDepth = hdr, 10
			if _, err := Compile(project, media, caps); err == nil || !strings.Contains(err.Error(), "this spike supports SDR encoding") {
				t.Fatalf("HDR metadata could be silently lost: %v", err)
			}
		})
	}
}

func contains(args []string, value string) bool {
	for _, arg := range args {
		if arg == value {
			return true
		}
	}
	return false
}
