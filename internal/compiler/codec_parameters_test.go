package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pheebkit/internal/core"
)

func TestCompileRejectsCodecFileAccessAndImplicitPassFiles(t *testing.T) {
	cases := []struct {
		option string
		value  string
	}{
		{"-x264-params", "pass=1"}, // Writes default statistics even without a path.
		{"-x264-params", "pass=2"},
		{"-x264-params", "stats=side-file"},
		{"-x264-params", "dump-yuv=side-file"},
		{"-x264-params", "cqmfile=side-file"},
		{"-x264-params", "cqm=side-file"},
		{"-x264-params", "opencl=1"}, // May create a cached kernel binary.
		{"-x264-params", "opencl-clbin=side-file"},
		{"-x265-params", "analysis-save=side-file"},
		{"-x265-params", "analysis-load=side-file"},
		{"-x265-params", "analysis-reuse-file=side-file"},
		{"-x265-params", "pass=1"},
		{"-x265-params", "pass=2"},
		{"-x265-params", "stats=side-file"},
		{"-x265-params", "multi-pass-opt-analysis=1"},
		{"-x265-params", "multi-pass-opt-distortion=1"},
		{"-x265-params", "csv=side-file"},
		{"-x265-params", "lambda-file=side-file"},
		{"-x265-params", "scaling-list=side-file"},
		{"-x265-params", "dhdr10-info=side-file"},
		{"-x265-params", "nalu-file=side-file"},
		{"-x265-params", "qpfile=side-file"},
		{"-x265-params", "svt=1"},
		{"-svtav1-params", "passes=2"},
		{"-svtav1-params", "pass=1"},
		{"-svtav1-params", "stats=side-file"},
		{"-svtav1-params", "fgs-table=side-file"},
		{"-svtav1-params", "recon=side-file"},
		{"-svtav1-params", "output=side-file"},
	}
	for _, test := range cases {
		t.Run(test.option+"/"+test.value, func(t *testing.T) {
			project, media, caps := fixture(t)
			selectCPUForRawOption(&project, &caps, test.option)
			project.Settings.RawArguments = []string{test.option, test.value}
			if _, err := Compile(project, media, caps); err == nil || !strings.Contains(err.Error(), "文件读写") {
				t.Fatalf("codec side-file access was accepted: %v", err)
			}
		})
	}
	// A real absolute path to an existing file is rejected at plan compilation,
	// before x265 can write its .temp file and replace that destination.
	project, media, caps := fixture(t)
	selectCPUForRawOption(&project, &caps, "-x265-params")
	sideFile := filepath.Join(filepath.Dir(project.Output.URI), "保留 analysis.bin")
	const original = "existing user content"
	if err := os.WriteFile(sideFile, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	project.Settings.RawArguments = []string{"-x265-params", "aq-mode=3:analysis-save=" + sideFile}
	if _, err := Compile(project, media, caps); err == nil {
		t.Fatal("existing side file could be overwritten")
	}
	contents, err := os.ReadFile(sideFile)
	if err != nil || string(contents) != original {
		t.Fatalf("existing side file changed: %q %v", contents, err)
	}
	if _, err := os.Stat(sideFile + ".temp"); !os.IsNotExist(err) {
		t.Fatalf("unexpected codec temporary file: %v", err)
	}
}

func TestCodecParametersRejectParserAmbiguity(t *testing.T) {
	for _, value := range []string{
		"AQ-MODE=3", "aq_mode=3", "aq-mode =3", "aq-mode= 3", "aq-mode=3 ",
		"aq-mode", "aq-mode=", "aq-mode=3=pass=1", "aq-mode=3:aq-mode=2",
		"aq-mode=3:", ":aq-mode=3", "aq-mode=3::rd=4", "aq-mode=3;pass=1",
		`aq-mode=3\:analysis-save=side-file`, `aq\-mode=3`, `'aq-mode'=3`, `aq-mode='3'`,
		"aq-mode=3\npass=1", "aq-mode=３", "aq-mode＝3", "aq-mode=3：pass=1",
		"aq-mode=3\x00:pass=1", "aq-mode=+3", "aq-mode=0x3", "aq-mode=3.0",
		"aq-mode=3e0", "aq-strength=NaN", "aq-strength=Inf", "aq-strength=1e0",
		"aq-strength=1/1", "aq-strength=1..0", "aq-strength=1,0", "aq-mode=999",
		"aq-strength=-1", "aq-mode=3:future-unknown-option=1", "crf=0", "bitrate=500", "qp=0",
	} {
		t.Run(value, func(t *testing.T) {
			if err := validateRaw([]string{"-x265-params", value}); err == nil {
				t.Fatalf("ambiguous codec value accepted: %q", value)
			}
		})
	}
	if err := validateRaw([]string{"-x265-params", "aq-strength=" + strings.Repeat("0", 4096)}); err == nil {
		t.Fatal("unbounded codec parameter accepted")
	}
}

func TestCompilePreservesSafeCodecControlsAndEncoderOwnership(t *testing.T) {
	cases := []struct {
		option string
		value  string
	}{
		{"-x264-params", "aq-mode=3:aq-strength=1.2:psy-rd=1.0,0.15:deblock=-1,-1:me=hex:mbtree=1"},
		{"-x265-params", "aq-mode=3:aq-strength=1.2:psy-rd=2.0:psy-rdoq=1.0:rd=4:deblock=-1,-1:log-level=error"},
		{"-svtav1-params", "tune=0:aq-mode=2:film-grain=8:film-grain-denoise=0:enable-tf=1:qm-min=0:qm-max=15"},
	}
	for _, test := range cases {
		t.Run(test.option, func(t *testing.T) {
			project, media, caps := fixture(t)
			selectCPUForRawOption(&project, &caps, test.option)
			media.SourceFingerprint = "verified-source-identity"
			project.Settings.RawArguments = []string{test.option, test.value, "-metadata", "title=中文 ' $ & ; : ="}
			plan, err := Compile(project, media, caps)
			if err != nil {
				t.Fatal(err)
			}
			if !hasArguments(plan.Args, project.Settings.RawArguments) || plan.SourceFingerprint != media.SourceFingerprint {
				t.Fatalf("validated parameters/source identity were changed: %+v", plan)
			}
			if err := validateEncoderRaw(project.Settings.RawArguments, "hevc_nvenc"); err == nil {
				t.Fatal("CPU codec parameters accepted on a GPU encoder")
			}
		})
	}
}

func selectCPUForRawOption(project *core.Project, caps *core.Capabilities, option string) {
	codecs := map[string]string{"-x264-params": "h264", "-x265-params": "hevc", "-svtav1-params": "av1"}
	encoders := map[string]string{"-x264-params": "libx264", "-x265-params": "libx265", "-svtav1-params": "libsvtav1"}
	project.Settings.Codec, project.Settings.Encoder = codecs[option], "cpu"
	caps.Encoders = []core.Encoder{{Name: encoders[option]}, {Name: "aac"}}
}
