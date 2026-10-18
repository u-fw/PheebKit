package compiler_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"pheebkit/internal/capabilities"
	"pheebkit/internal/compiler"
	"pheebkit/internal/core"
	windowsrunner "pheebkit/internal/runner/windows"
)

func TestIntegrationSegmentsSubtitleBurnAndCatalog(t *testing.T) {
	if os.Getenv("PHEEBKIT_INTEGRATION") != "1" {
		t.Skip("real FFmpeg opt-in")
	}
	ffmpeg, ffprobe := os.Getenv("PHEEBKIT_FFMPEG"), os.Getenv("PHEEBKIT_FFPROBE")
	if ffmpeg == "" {
		ffmpeg = "ffmpeg.exe"
	}
	if ffprobe == "" {
		ffprobe = "ffprobe.exe"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	dir := t.TempDir()
	run := func(args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(ctx, ffmpeg, args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out, err := cmd.Output()
		if err != nil {
			var stderr []byte
			if exit, ok := err.(*exec.ExitError); ok {
				stderr = exit.Stderr
			}
			t.Fatalf("FFmpeg: %v %s", err, stderr)
		}
		return out
	}
	sub := filepath.Join(dir, "字幕 '[逗号,;].srt")
	if err := os.WriteFile(sub, []byte("1\n00:00:01,000 --> 00:00:01,900\nPheebKit\n\n2\n00:00:02,100 --> 00:00:03,800\nSecond\n"), 0600); err != nil {
		t.Fatal(err)
	}
	input := core.Source{URI: filepath.Join(dir, "source 中文.mkv"), Kind: core.SourceLocalFile}
	run("-hide_banner", "-loglevel", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "color=red:size=320x180:rate=10:d=2", "-f", "lavfi", "-i", "color=blue:size=320x180:rate=10:d=2", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=4", "-f", "srt", "-i", sub, "-filter_complex", "[0:v][1:v]concat=n=2:v=1:a=0[v]", "-map", "[v]", "-map", "2:a", "-map", "3:s", "-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac", "-c:s", "srt", input.URI)
	runner := windowsrunner.New(ffmpeg, ffprobe)
	media, err := runner.Probe(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	caps := core.Capabilities{Encoders: capabilities.ParseEncoders(string(run("-hide_banner", "-encoders")))}
	for _, category := range []string{"encoders", "decoders", "filters", "muxers", "demuxers", "protocols", "bsfs", "pixel-formats", "sample-formats", "layouts", "colors", "hardware", "devices"} {
		entries, err := runner.Catalog(ctx, category)
		if err != nil {
			t.Fatalf("%s: %v", category, err)
		}
		if category != "devices" && len(entries) == 0 {
			t.Fatalf("empty %s", category)
		}
		if category == "filters" {
			for _, e := range entries {
				caps.Filters = append(caps.Filters, e.Name)
			}
		}
		t.Logf("%s: %d entries", category, len(entries))
	}
	help, err := runner.OptionHelp(ctx, "filters", "subtitles")
	if err != nil || !strings.Contains(help, "force_style") {
		t.Fatal("local help missing", err)
	}
	full, err := runner.OptionHelp(ctx, "full", "")
	if err != nil || !strings.Contains(full, "-filter_complex") {
		t.Fatal("full local option help unavailable", err)
	}
	if _, err := runner.OptionHelp(ctx, "filters", "subtitles;movie=file"); err == nil {
		t.Fatal("unsafe help name accepted")
	}
	p := core.Project{SchemaVersion: 1, ID: "segments", Inputs: []core.Source{input}, Output: core.Source{URI: filepath.Join(dir, "joined.mp4"), Kind: core.SourceLocalFile}, Settings: core.DefaultSettings()}
	p.Settings.Codec, p.Settings.Encoder, p.Settings.Preset = "h264", "cpu", "ultrafast"
	p.Settings.Segments = []core.TimeRange{{StartSeconds: 2, EndSeconds: 3}, {StartSeconds: 0, EndSeconds: 1}}
	p.Settings.VolumeDB = -6
	execute := func(project core.Project) core.Media {
		t.Helper()
		plan, err := compiler.Compile(project, media, caps)
		if err != nil {
			t.Fatal(err)
		}
		plan.JobID = filepath.Base(project.Output.URI)
		if _, err := runner.Start(ctx, plan, core.Callbacks{}); err != nil {
			t.Fatal(err)
		}
		out, err := runner.Probe(ctx, project.Output)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	joined := execute(p)
	if joined.DurationSeconds < 1.9 || joined.DurationSeconds > 2.2 {
		t.Fatal("wrong joined duration", joined.DurationSeconds)
	}
	rms := func(path string) float64 {
		data := run("-hide_banner", "-loglevel", "error", "-ss", "0.3", "-i", path, "-map", "0:a:0", "-t", "0.2", "-f", "s16le", "-acodec", "pcm_s16le", "pipe:1")
		sum := 0.0
		for i := 0; i+1 < len(data); i += 2 {
			v := float64(int16(binary.LittleEndian.Uint16(data[i:])))
			sum += v * v
		}
		return math.Sqrt(sum / float64(len(data)/2))
	}
	if ratio := rms(p.Output.URI) / rms(input.URI); ratio < .4 || ratio > .65 {
		t.Fatal("joined audio volume not applied", ratio)
	}
	frame := func(path string, seconds float64) []byte {
		return run("-hide_banner", "-loglevel", "error", "-ss", fmt.Sprint(seconds), "-i", path, "-frames:v", "1", "-f", "rawvideo", "-pix_fmt", "rgb24", "pipe:1")
	}
	for _, point := range []struct {
		time    float64
		channel int
	}{{.4, 2}, {1.4, 0}} {
		data := frame(p.Output.URI, point.time)
		sums := [3]int{}
		for i, v := range data {
			sums[i%3] += int(v)
		}
		if sums[point.channel] < sums[2-point.channel]*3 {
			t.Fatal("segment order wrong", point.time, sums)
		}
	}
	p.Settings.Segments = nil
	p.Settings.VolumeDB = 0
	p.Settings.Trim = &core.TimeRange{StartSeconds: 1.2, EndSeconds: 1.8}
	p.Settings.SubtitleBurn = &core.SubtitleBurn{Source: "external", Path: sub, Font: "Arial", FontSize: 28, Color: "#ffffff", OutlineColor: "#000000", Outline: 1, Alignment: 2, Margin: 10}
	p.Output.URI = filepath.Join(dir, "burned.mp4")
	execute(p)
	white := func(path string) int {
		count := 0
		data := frame(path, .2)
		for i := 0; i+2 < len(data); i += 3 {
			if data[i] > 180 && data[i+1] > 180 && data[i+2] > 180 {
				count++
			}
		}
		return count
	}
	if white(p.Output.URI) < 20 {
		t.Fatal("external subtitle not rendered at nonzero trim")
	}
	p.Output.URI = filepath.Join(dir, "offset.mp4")
	p.Settings.SubtitleBurn.OffsetSeconds = 1
	execute(p)
	if white(p.Output.URI) > 10 {
		t.Fatal("positive subtitle delay was not applied")
	}
	p.Output.URI = filepath.Join(dir, "embedded.mp4")
	p.Settings.SubtitleBurn.Source = "embedded"
	p.Settings.SubtitleBurn.Path = ""
	p.Settings.SubtitleBurn.StreamIndex = 2
	p.Settings.SubtitleBurn.OffsetSeconds = 0
	execute(p)
	if white(p.Output.URI) < 20 {
		t.Fatal("embedded subtitle not rendered")
	}
	p.Settings.Trim = nil
	p.Settings.Segments = []core.TimeRange{{StartSeconds: 2.2, EndSeconds: 2.8}, {StartSeconds: 1.2, EndSeconds: 1.8}}
	p.Output.URI = filepath.Join(dir, "joined-burn.mp4")
	execute(p)
	if white(p.Output.URI) < 20 {
		t.Fatal("subtitle did not follow reordered segment")
	}
	p.Settings.Audio = "none"
	p.Output.URI = filepath.Join(dir, "joined-silent.mp4")
	silent := execute(p)
	for _, stream := range silent.Streams {
		if stream.CodecType == "audio" {
			t.Fatal("silent concatenation retained audio")
		}
	}
	p.Settings.Segments = nil
	p.Settings.Trim = &core.TimeRange{StartSeconds: 1.2, EndSeconds: 1.8}
	p.Settings.SubtitleBurn.Source = "external"
	p.Settings.SubtitleBurn.Path = sub
	p.Output.URI = filepath.Join(dir, "mutated.mp4")
	plan, err := compiler.Compile(p, media, caps)
	if err != nil {
		t.Fatal(err)
	}
	plan.JobID = "changed-subtitle"
	if err := os.WriteFile(sub, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Start(ctx, plan, core.Callbacks{}); err == nil {
		t.Fatal("changed subtitle accepted")
	}
	if _, err := os.Stat(p.Output.URI); !os.IsNotExist(err) {
		t.Fatal("failed output was published")
	}
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { hits.Add(1); http.Error(w, "unexpected", 404) }))
	defer server.Close()
	malicious := filepath.Join(dir, "playlist-disguised.srt")
	if err := os.WriteFile(malicious, []byte("#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\n"+server.URL+"/segment.ts\n#EXT-X-ENDLIST\n"), 0600); err != nil {
		t.Fatal(err)
	}
	p.Settings.SubtitleBurn.Path = malicious
	p.Output.URI = filepath.Join(dir, "unsafe-subtitle.mp4")
	unsafePlan, err := compiler.Compile(p, media, caps)
	if err != nil {
		t.Fatal(err)
	}
	unsafePlan.JobID = "unsafe-subtitle"
	if _, err := runner.Start(ctx, unsafePlan, core.Callbacks{}); err == nil {
		t.Fatal("disguised playlist accepted as text subtitle")
	}
	if hits.Load() != 0 {
		t.Fatal("subtitle preparation contacted HTTP")
	}
	p.Settings.SubtitleBurn.Source = "embedded"
	p.Settings.SubtitleBurn.Path = ""
	p.Output.URI = filepath.Join(dir, "cancelled-subtitle.mp4")
	cancelPlan, err := compiler.Compile(p, media, caps)
	if err != nil {
		t.Fatal(err)
	}
	cancelPlan.JobID = "cancel-subtitle"
	cancelCtx, stop := context.WithCancel(ctx)
	stop()
	if _, err := runner.Start(cancelCtx, cancelPlan, core.Callbacks{}); err == nil {
		t.Fatal("cancelled subtitle preparation succeeded")
	}
	if _, err := os.Stat(p.Output.URI); !os.IsNotExist(err) {
		t.Fatal("cancelled subtitle output published")
	}
}
