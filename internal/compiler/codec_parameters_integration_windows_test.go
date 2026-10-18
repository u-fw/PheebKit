package compiler

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"pheebkit/internal/core"
)

// Execute the validated argument vector against the installed libraries. This
// catches allowlist spelling/value mistakes that unit validation cannot find.
func TestIntegrationSafeCodecParametersEncodeWithoutSideFiles(t *testing.T) {
	if os.Getenv("PHEEBKIT_INTEGRATION") != "1" {
		t.Skip("set PHEEBKIT_INTEGRATION=1 to execute installed FFmpeg/FFprobe")
	}
	ffmpeg := integrationCodecTool(t, "PHEEBKIT_FFMPEG", "ffmpeg.exe")
	ffprobe := integrationCodecTool(t, "PHEEBKIT_FFPROBE", "ffprobe.exe")
	dir := t.TempDir()
	input := filepath.Join(dir, "输入 sample.mkv")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	run := func(binary string, args ...string) []byte {
		t.Helper()
		command := exec.CommandContext(ctx, binary, args...)
		command.Dir = dir
		command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		data, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("real codec command failed: %v\n%s", err, data)
		}
		return data
	}
	run(ffmpeg, "-hide_banner", "-loglevel", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "testsrc2=size=128x96:rate=10", "-frames:v", "3", "-c:v", "libx264", "-preset", "ultrafast", input)
	inputInfo, err := os.Stat(input)
	if err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(dir, "preserved analysis.bin")
	const original = "existing data must survive codec configuration"
	if err := os.WriteFile(sentinel, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	media := core.Media{Source: core.Source{URI: input, Kind: core.SourceLocalFile}, SizeBytes: inputInfo.Size(), DurationSeconds: 0.3,
		Streams: []core.Stream{{Index: 0, CodecType: "video", CodecName: "h264", Width: 128, Height: 96, BitDepth: 8}}}
	for _, test := range []struct {
		option string
		value  string
		codec  string
	}{
		{"-x264-params", "aq-mode=3:aq-strength=1.2:psy-rd=1.0,0.15:deblock=-1,-1:me=hex:mbtree=1", "h264"},
		{"-x265-params", "aq-mode=3:aq-strength=1.2:psy-rd=2.0:psy-rdoq=1.0:rd=4:deblock=-1,-1:log-level=error", "hevc"},
		{"-svtav1-params", "tune=0:aq-mode=2:film-grain=8:film-grain-denoise=0:enable-tf=1:enable-qm=1:qm-min=0:qm-max=15", "av1"},
	} {
		t.Run(test.codec, func(t *testing.T) {
			project := core.Project{SchemaVersion: core.ProjectSchemaVersion, ID: "safe-codec-integration", Inputs: []core.Source{media.Source},
				Output: core.Source{URI: filepath.Join(dir, test.codec+".mp4"), Kind: core.SourceLocalFile}, Settings: core.DefaultSettings()}
			caps := core.Capabilities{}
			selectCPUForRawOption(&project, &caps, test.option)
			project.Settings.Preset = "fast"
			project.Settings.RawArguments = []string{test.option, test.value}
			plan, err := Compile(project, media, caps)
			if err != nil {
				t.Fatal(err)
			}
			data := run(ffmpeg, plan.Args...)
			if strings.Contains(string(data), "Error parsing option") || strings.Contains(string(data), "Unknown option") {
				t.Fatalf("safe controls were silently ignored by the library:\n%s", data)
			}
			data = run(ffprobe, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_name,width,height", "-of", "json", project.Output.URI)
			var probe struct {
				Streams []struct {
					CodecName string `json:"codec_name"`
					Width     int    `json:"width"`
					Height    int    `json:"height"`
				} `json:"streams"`
			}
			if err := json.Unmarshal(data, &probe); err != nil || len(probe.Streams) != 1 || probe.Streams[0].CodecName != test.codec || probe.Streams[0].Width != 128 || probe.Streams[0].Height != 96 {
				t.Fatalf("safe controls did not produce the expected video: %s %v", data, err)
			}
			t.Logf("%s controls encoded a verified 128x96 %s output", test.option, test.codec)
		})
	}
	contents, err := os.ReadFile(sentinel)
	if err != nil || string(contents) != original {
		t.Fatalf("unrelated side file changed: %q %v", contents, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	allowedFiles := map[string]bool{"输入 sample.mkv": true, "preserved analysis.bin": true, "h264.mp4": true, "hevc.mp4": true, "av1.mp4": true}
	for _, entry := range entries {
		if !allowedFiles[entry.Name()] {
			t.Fatalf("codec created an unmanaged side file: %s", entry.Name())
		}
	}
}

func integrationCodecTool(t *testing.T, environment, name string) string {
	t.Helper()
	if configured := os.Getenv(environment); configured != "" {
		if info, err := os.Stat(configured); err == nil && info.IsDir() {
			return filepath.Join(configured, name)
		}
		return configured
	}
	path, err := exec.LookPath(name)
	if err != nil {
		t.Fatal(err)
	}
	return path
}
