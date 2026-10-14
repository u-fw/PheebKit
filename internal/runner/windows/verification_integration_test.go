package windows

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"kompeg/internal/compiler"
	"kompeg/internal/core"
)

func TestIntegrationPostflightUtilityOutputs(t *testing.T) {
	if os.Getenv("KOMPEG_INTEGRATION") != "1" {
		t.Skip("set KOMPEG_INTEGRATION=1 to execute installed FFmpeg/FFprobe")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	runner := New(os.Getenv("KOMPEG_FFMPEG"), os.Getenv("KOMPEG_FFPROBE"))
	dir := t.TempDir()
	subtitle := filepath.Join(dir, "input.srt")
	if err := os.WriteFile(subtitle, []byte("1\n00:00:00,100 --> 00:00:01,500\nPostflight 字幕\n"), 0600); err != nil {
		t.Fatal(err)
	}
	input := core.Source{Kind: core.SourceLocalFile, URI: filepath.Join(dir, "input.mkv")}
	cmd := exec.CommandContext(ctx, runner.FFmpeg, "-hide_banner", "-loglevel", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "testsrc2=size=160x90:rate=10:duration=2", "-f", "lavfi", "-i", "sine=duration=2", "-i", subtitle, "-map", "0:v", "-map", "1:a", "-map", "2:s", "-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac", "-c:s", "subrip", input.URI)
	hideProcess(cmd)
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate fixture: %v %s", err, data)
	}
	media, err := runner.Probe(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	caps := core.Capabilities{Encoders: []core.Encoder{{Name: "pcm_s16le", MediaType: "audio"}, {Name: "subrip", MediaType: "subtitle"}}}
	for _, item := range []struct {
		operation, ext, audio string
		counts                [3]int
	}{
		{"remux", ".mkv", "copy", [3]int{1, 1, 1}},
		{"extract-audio", ".wav", "wav", [3]int{0, 1, 0}},
		{"extract-subtitle", ".srt", "none", [3]int{0, 0, 1}},
	} {
		t.Run(item.operation, func(t *testing.T) {
			settings := core.DefaultSettings()
			settings.Operation, settings.Audio = item.operation, item.audio
			settings.AudioSelection = &core.TrackSelection{Mode: "none"}
			settings.SubtitleSelection = &core.TrackSelection{Mode: "none"}
			if item.counts[1] > 0 {
				settings.AudioSelection = &core.TrackSelection{Mode: "selected", Indices: []int{1}}
			}
			if item.counts[2] > 0 {
				settings.SubtitleSelection = &core.TrackSelection{Mode: "selected", Indices: []int{2}}
			}
			project := core.Project{SchemaVersion: 1, ID: item.operation, Inputs: []core.Source{input}, Output: core.Source{Kind: core.SourceLocalFile, URI: filepath.Join(dir, item.operation+item.ext)}, Settings: settings}
			plan, err := compiler.Compile(project, media, caps)
			if err != nil {
				t.Fatal(err)
			}
			plan.JobID = item.operation
			result, err := runner.Start(ctx, plan, core.Callbacks{})
			if err != nil {
				t.Fatal(err)
			}
			v := result.Verification
			if v == nil || [3]int{v.VideoStreams, v.AudioStreams, v.SubtitleStreams} != item.counts || len(v.Warnings) != 0 {
				t.Fatalf("unexpected postflight: %+v", v)
			}
		})
	}
}
