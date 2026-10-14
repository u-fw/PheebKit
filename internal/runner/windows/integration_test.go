package windows

import (
	"context"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"kompeg/internal/compiler"
	"kompeg/internal/core"
)

// Run explicitly: $env:KOMPEG_INTEGRATION='1'; go test ./internal/runner/windows -run Integration -v
func TestIntegrationRealHEVCProbeProgressAndCancellation(t *testing.T) {
	if os.Getenv("KOMPEG_INTEGRATION") != "1" {
		t.Skip("set KOMPEG_INTEGRATION=1 to execute installed FFmpeg/FFprobe")
	}
	runner := New(os.Getenv("KOMPEG_FFMPEG"), os.Getenv("KOMPEG_FFPROBE"))
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "输入 video & ;.mkv")
	input := core.Source{URI: inputPath, DisplayName: filepath.Base(inputPath), Kind: core.SourceLocalFile}
	generate := exec.CommandContext(ctx, runner.FFmpeg, "-hide_banner", "-nostdin", "-n", "-f", "lavfi", "-i", "testsrc2=size=640x360:rate=30:duration=6", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=6", "-c:v", "libx264", "-preset", "ultrafast", "-crf", "18", "-c:a", "flac", "-shortest", inputPath)
	hideProcess(generate)
	if data, err := generate.CombinedOutput(); err != nil {
		t.Fatalf("generate synthetic fixture: %v\n%s", err, data)
	}
	media, err := runner.Probe(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(media.Streams) != 2 || media.DurationSeconds < 5.9 || media.SizeBytes == 0 || media.Streams[0].Width != 640 {
		t.Fatalf("unexpected input probe: %+v", media)
	}
	caps, err := runner.Capabilities(ctx)
	if err != nil {
		t.Fatal(err)
	}
	project := core.Project{SchemaVersion: 1, ID: "integration", Inputs: []core.Source{input},
		Output: core.Source{URI: filepath.Join(dir, "输出 ' $ &.mp4"), Kind: core.SourceLocalFile}, Settings: core.DefaultSettings()}
	project.Settings.Quality, project.Settings.Preset = 28, "fast"
	project.Settings.Encoder = "cpu"
	plan, err := compiler.Compile(project, media, caps)
	if err != nil {
		t.Fatal(err)
	}
	plan.JobID = "integration-hevc"
	var progressCount, logCount atomic.Int32
	result, err := runner.Start(ctx, plan, core.Callbacks{
		Progress: func(progress core.Progress) {
			progressCount.Add(1)
			if progress.Percent < 0 || progress.Percent > 100 {
				t.Errorf("invalid progress percent %f", progress.Percent)
			}
		},
		Log: func(string) { logCount.Add(1) },
	})
	if err != nil {
		t.Fatal(err)
	}
	output, err := runner.Probe(ctx, project.Output)
	if err != nil {
		t.Fatal(err)
	}
	if output.Streams[0].CodecName != "hevc" || math.Abs(output.DurationSeconds-media.DurationSeconds) > 0.2 || result.OutputSizeBytes <= 0 || result.InputSizeBytes != media.SizeBytes || result.ElapsedSeconds <= 0 || result.AverageSpeed <= 0 {
		t.Fatalf("invalid real encoding result: output=%+v result=%+v", output, result)
	}
	if progressCount.Load() == 0 || logCount.Load() == 0 {
		t.Fatalf("progress/stderr separation failed: progress=%d log=%d", progressCount.Load(), logCount.Load())
	}
	if result.Verification == nil || result.Verification.VideoStreams != 1 || result.Verification.AudioStreams != 1 || len(result.Verification.Warnings) != 0 {
		t.Fatalf("real output was not structurally verified: %+v", result.Verification)
	}
	t.Run("failed and cancelled postflight never publish", func(t *testing.T) {
		for _, mode := range []string{"missing-track", "probe-unavailable", "cancel-verification"} {
			t.Run(mode, func(t *testing.T) {
				candidate := plan
				candidate.JobID = "postflight-" + mode
				candidate.Output.URI = filepath.Join(dir, mode+".mp4")
				candidate.Args = append([]string{}, plan.Args...)
				candidate.Args[len(candidate.Args)-1] = candidate.Output.URI
				adapter := New(runner.FFmpeg, runner.FFprobe)
				if mode == "missing-track" {
					candidate.ExpectedOutput = &core.OutputExpectation{VideoStreams: 1, AudioStreams: 2}
				} else if mode == "probe-unavailable" {
					adapter.FFprobe = filepath.Join(dir, "missing-ffprobe.exe")
				}
				callbacks := core.Callbacks{}
				if mode == "cancel-verification" {
					callbacks.Log = func(line string) {
						if line == "正在校验输出结构…" {
							if err := adapter.Cancel(candidate.JobID); err != nil {
								t.Error(err)
							}
						}
					}
				}
				if _, err := adapter.Start(ctx, candidate, callbacks); err == nil {
					t.Fatal("invalid/cancelled postflight reported success")
				}
				if _, err := os.Stat(candidate.Output.URI); !os.IsNotExist(err) {
					t.Fatalf("failed postflight published output: %v", err)
				}
				matches, err := filepath.Glob(filepath.Join(dir, ".kompeg-*"))
				if err != nil || len(matches) != 0 {
					t.Fatalf("scratch files leaked: %v %v", matches, err)
				}
			})
		}
	})
	t.Logf("FFmpeg=%s; HEVC input=%d output=%d saved=%.2f%% elapsed=%.2fs speed=%.2fx updates=%d logs=%d", runner.FFmpeg, result.InputSizeBytes, result.OutputSizeBytes, result.SavedPercent, result.ElapsedSeconds, result.AverageSpeed, progressCount.Load(), logCount.Load())

	t.Run("CPU preserves genuine 10-bit input", func(t *testing.T) {
		tenBitSource := core.Source{URI: filepath.Join(dir, "CPU 10-bit input.mp4"), Kind: core.SourceLocalFile}
		generate10 := exec.CommandContext(ctx, runner.FFmpeg, "-hide_banner", "-loglevel", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "testsrc2=size=640x360:rate=30:duration=2", "-c:v", "libx265", "-preset", "ultrafast", "-x265-params", "log-level=error", "-crf", "18", "-pix_fmt", "yuv420p10le", tenBitSource.URI)
		hideProcess(generate10)
		if data, err := generate10.CombinedOutput(); err != nil {
			t.Fatalf("generate genuine 10-bit fixture: %v\n%s", err, data)
		}
		tenBitMedia, err := runner.Probe(ctx, tenBitSource)
		if err != nil || len(tenBitMedia.Streams) != 1 || tenBitMedia.Streams[0].BitDepth != 10 {
			t.Fatalf("fixture is not genuine 10-bit: %+v %v", tenBitMedia, err)
		}
		tenBitProject := project
		tenBitProject.Inputs = []core.Source{tenBitSource}
		tenBitProject.Output = core.Source{URI: filepath.Join(dir, "CPU 10-bit output.mp4"), Kind: core.SourceLocalFile}
		tenBitProject.Settings.Resolution = 720
		tenBitPlan, err := compiler.Compile(tenBitProject, tenBitMedia, caps)
		if err != nil {
			t.Fatal(err)
		}
		tenBitPlan.JobID = "integration-cpu-ten-bit"
		if _, err := runner.Start(ctx, tenBitPlan, core.Callbacks{}); err != nil {
			t.Fatal(err)
		}
		encoded, err := runner.Probe(ctx, tenBitProject.Output)
		if err != nil || len(encoded.Streams) != 1 || encoded.Streams[0].BitDepth != 10 || encoded.Streams[0].Height != 720 || encoded.Streams[0].CodecName != "hevc" {
			t.Fatalf("CPU conversion lost depth or failed resize: %+v %v", encoded, err)
		}
	})

	t.Run("cancel real process", func(t *testing.T) {
		project.Output.URI = filepath.Join(dir, "cancelled output.mp4")
		plan, err := compiler.Compile(project, media, caps)
		if err != nil {
			t.Fatal(err)
		}
		plan.JobID = "integration-cancel"
		for i, arg := range plan.Args {
			if arg == "-i" {
				plan.Args = append(append(append([]string{}, plan.Args[:i]...), "-re", "-stats_period", "0.1"), plan.Args[i:]...)
				break
			}
		}
		progress := make(chan struct{}, 1)
		done := make(chan error, 1)
		go func() {
			_, err := runner.Start(ctx, plan, core.Callbacks{Progress: func(core.Progress) {
				select {
				case progress <- struct{}{}:
				default:
				}
			}})
			done <- err
		}()
		select {
		case <-progress:
		case err := <-done:
			t.Fatalf("process exited before cancellation: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("cancel fixture produced no progress")
		}
		started := time.Now()
		if err := runner.Cancel(plan.JobID); err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("cancel returned %v", err)
			}
			if time.Since(started) > 3*time.Second {
				t.Fatal("encoder process did not promptly exit")
			}
		case <-time.After(4 * time.Second):
			t.Fatalf("cancel left FFmpeg or launcher running: %s", strings.Join(plan.Args, " "))
		}
		if _, err := os.Stat(project.Output.URI); !os.IsNotExist(err) {
			t.Fatalf("cancellation left a partial requested output: %v", err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".kompeg-") {
				t.Fatalf("cancellation leaked scratch directory %s", entry.Name())
			}
		}
		plan.JobID = "integration-retry"
		// Retry the same output path without -re to prove cancellation released it.
		plan, err = compiler.Compile(project, media, caps)
		if err != nil {
			t.Fatal(err)
		}
		plan.JobID = "integration-retry"
		if _, err := runner.Start(ctx, plan, core.Callbacks{}); err != nil {
			t.Fatalf("retry after cancellation failed: %v", err)
		}
	})
}
