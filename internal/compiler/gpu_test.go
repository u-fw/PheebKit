package compiler

import (
	"testing"

	"kompeg/internal/core"
)

func TestGPUPlansUseVendorRateControlAndPresets(t *testing.T) {
	cases := []struct {
		name    string
		quality []string
		bitrate []string
	}{
		{"hevc_nvenc", []string{"-preset", "p5", "-rc", "vbr", "-cq", "26", "-b:v", "0"}, []string{"-rc", "vbr", "-b:v", "3728k"}},
		{"hevc_qsv", []string{"-preset", "medium", "-global_quality", "26"}, []string{"-b:v", "3728k"}},
		{"hevc_amf", []string{"-quality", "balanced", "-rc", "cqp", "-qp_i", "26", "-qp_p", "26"}, []string{"-rc", "vbr_peak", "-maxrate", "7456k", "-b:v", "3728k"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			project, media, caps := fixture(t)
			caps.VideoEncoders = []core.VideoEncoder{{Name: test.name, Codec: "hevc", Label: "GPU", Kind: "gpu", Available: true, PixelFormats: []string{"nv12", "p010le"}}}
			plan, err := Compile(project, media, caps)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Encoder != test.name || plan.EncoderKind != "gpu" || contains(plan.Args, "-crf") || !hasArguments(plan.Args, test.quality) || !hasArguments(plan.Args, []string{"-vf", "scale=iw:ih,format=nv12", "-pix_fmt", "+nv12"}) {
				t.Fatalf("wrong GPU quality plan: %+v", plan)
			}
			project.Settings.Mode, project.Settings.TargetSizeMB = "size", 5
			media.Streams[0].BitDepth = 10
			project.Settings.Resolution = 720
			plan, err = Compile(project, media, caps)
			if err != nil || !hasArguments(plan.Args, test.bitrate) || !hasArguments(plan.Args, []string{"-vf", "scale=-2:720,format=p010le", "-pix_fmt", "+p010le"}) || contains(plan.Args, "-crf") {
				t.Fatalf("wrong GPU size/depth plan: %q %v", plan.Args, err)
			}
		})
	}
}

func TestSoftwareRawParametersAreNeverSilentlyIgnoredByGPU(t *testing.T) {
	project, media, caps := fixture(t)
	caps.VideoEncoders = []core.VideoEncoder{{Name: "hevc_nvenc", Codec: "hevc", Kind: "gpu", Available: true, PixelFormats: []string{"nv12"}}}
	project.Settings.RawArguments = []string{"-x265-params", "aq-mode=3"}
	if _, err := Compile(project, media, caps); err == nil {
		t.Fatal("CPU-only raw option was allowed with GPU encoder")
	}
	project.Settings.Encoder = "cpu"
	caps.VideoEncoders = nil
	if _, err := Compile(project, media, caps); err != nil {
		t.Fatalf("matching libx265 advanced arguments were rejected: %v", err)
	}
	project.Settings.RawArguments = []string{"-x264-params", "aq-mode=3"}
	if _, err := Compile(project, media, caps); err == nil {
		t.Fatal("software arguments for the wrong codec were accepted")
	}
}

func TestGPUQualityRangesAndAV1AMFScale(t *testing.T) {
	settings := core.Settings{Codec: "av1", Mode: "quality", Preset: "slow", Quality: 63}
	args, err := VideoArguments("av1_amf", settings, 0)
	if err != nil || !hasArguments(args, []string{"-quality", "quality", "-rc", "cqp", "-qp_i", "255", "-qp_p", "255", "-qp_b", "255"}) {
		t.Fatalf("AMF AV1 QP range was treated as CRF: %q %v", args, err)
	}
	args, err = VideoArguments("av1_nvenc", settings, 0)
	if err != nil || !hasArguments(args, []string{"-cq", "63"}) {
		t.Fatalf("NVENC AV1 native quality range rejected: %q %v", args, err)
	}
	if _, err := VideoArguments("av1_qsv", settings, 0); err == nil {
		t.Fatal("QSV ICQ accepted quality above 51")
	}
	settings.Codec, settings.Quality = "hevc", 0
	if _, err := VideoArguments("hevc_nvenc", settings, 0); err == nil {
		t.Fatal("automatic CQ 0 was advertised as best quality")
	}
	settings.Quality, settings.Preset = 26, "ultrafast"
	if _, err := VideoArguments("hevc_nvenc", settings, 0); err == nil {
		t.Fatal("CPU-only preset accepted by GPU")
	}
}

func hasArguments(args, expected []string) bool {
	// Each expected pair must occur together, independent of ordering.
	for index := 0; index < len(expected); index += 2 {
		found := false
		for actual := 0; actual+1 < len(args); actual++ {
			if args[actual] == expected[index] && args[actual+1] == expected[index+1] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
