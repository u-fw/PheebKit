package compiler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"kompeg/internal/capabilities"
	"kompeg/internal/compiler"
	"kompeg/internal/core"
	"kompeg/internal/ffmpegpolicy"
	windowsrunner "kompeg/internal/runner/windows"
)

func TestIntegrationCorruptRecovery(t *testing.T) {
	if os.Getenv("KOMPEG_INTEGRATION") != "1" {
		t.Skip("real FFmpeg opt-in")
	}
	ffmpeg, ffprobe := os.Getenv("KOMPEG_FFMPEG"), os.Getenv("KOMPEG_FFPROBE")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	run := func(binary string, args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		data, err := cmd.Output()
		if err != nil {
			t.Fatalf("%v: %s", err, data)
		}
		return data
	}
	dir := t.TempDir()
	src := core.Source{Kind: core.SourceLocalFile, URI: filepath.Join(dir, "corrupt.mp4")}
	run(ffmpeg, "-v", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "testsrc2=size=160x90:rate=10:duration=16", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=16", "-c:v", "libx264", "-preset", "ultrafast", "-g", "10", "-keyint_min", "10", "-sc_threshold", "0", "-c:a", "aac", src.URI)
	runner := windowsrunner.New(ffmpeg, ffprobe)
	clean, err := runner.Probe(ctx, src)
	if err != nil {
		t.Fatal(err)
	}
	report, err := runner.ScanHealth(ctx, clean, 5, nil)
	if err != nil || report.State != "completed" || len(report.DamagedRanges) != 0 {
		t.Fatalf("healthy scan: %+v %v", report, err)
	}
	var packets struct {
		Packets []struct {
			PTS  string `json:"pts_time"`
			Pos  string `json:"pos"`
			Size string `json:"size"`
		} `json:"packets"`
	}
	if err := json.Unmarshal(run(ffprobe, "-v", "error", "-select_streams", "v:0", "-show_packets", "-show_entries", "packet=pts_time,pos,size", "-of", "json", src.URI), &packets); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(src.URI)
	if err != nil {
		t.Fatal(err)
	}
	mutated := false
	for _, p := range packets.Packets {
		pts, _ := strconv.ParseFloat(p.PTS, 64)
		pos, _ := strconv.Atoi(p.Pos)
		size, _ := strconv.Atoi(p.Size)
		if pts >= 6.2 && pts < 6.4 && size > 64 {
			for i := pos + 12; i < pos+size; i++ {
				data[i] = 0x00
			}
			mutated = true
		}
	}
	if !mutated {
		t.Fatal("fixture packet missing")
	}
	if err := os.WriteFile(src.URI, data, 0600); err != nil {
		t.Fatal(err)
	}
	media, err := runner.Probe(ctx, src)
	if err != nil {
		t.Fatal(err)
	}
	report, err = runner.ScanHealth(ctx, media, 5, nil)
	if err != nil || len(report.DamagedRanges) == 0 {
		t.Fatalf("corrupt scan: %+v %v", report, err)
	}
	t.Logf("damaged windows: %+v", report.DamagedRanges)
	fine, err := runner.ScanHealth(ctx, media, 1, nil)
	if err != nil || len(fine.DamagedRanges) == 0 {
		t.Fatal("fine scan did not identify corruption", err)
	}
	if fine.WindowSeconds != 1 {
		t.Fatal("requested scan window ignored")
	}
	t.Logf("one-second windows: %+v", fine.DamagedRanges)
	if report.DamagedRanges[0].StartSeconds > 6.2 || report.DamagedRanges[0].EndSeconds < 6.4 {
		t.Fatal("scan missed mutated window")
	}
	caps := core.Capabilities{Encoders: capabilities.ParseEncoders(string(run(ffmpeg, "-hide_banner", "-encoders")))}
	entries, err := runner.Catalog(ctx, "filters")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range entries {
		caps.Filters = append(caps.Filters, f.Name)
	}
	for _, mode := range []string{"delete", "black"} {
		p := core.Project{SchemaVersion: 1, ID: mode, Inputs: []core.Source{src}, Output: core.Source{Kind: core.SourceLocalFile, URI: filepath.Join(dir, mode+".mp4")}, Settings: core.DefaultSettings()}
		p.Settings.Codec, p.Settings.Encoder, p.Settings.Preset = "h264", "cpu", "ultrafast"
		p.Settings.Recovery = &core.Recovery{Mode: mode, SourceFingerprint: media.SourceFingerprint, DamagedRanges: report.DamagedRanges}
		plan, err := compiler.Compile(p, media, caps)
		if err != nil {
			t.Fatal(err)
		}
		plan.JobID = mode
		result, err := runner.Start(ctx, plan, core.Callbacks{})
		if err != nil {
			t.Fatal(mode, err)
		}
		if result.Verification == nil || !result.Verification.FullDecodePassed {
			t.Fatal("missing full decode verification")
		}
		expected := media.DurationSeconds
		if mode == "delete" {
			for _, r := range report.DamagedRanges {
				expected -= r.EndSeconds - r.StartSeconds
			}
		}
		if math.Abs(result.Verification.DurationSeconds-expected) > .3 {
			t.Fatal("wrong recovered duration", mode, result.Verification.DurationSeconds, expected)
		}
		if mode == "black" {
			raw := run(ffmpeg, "-v", "error", "-ss", "6.5", "-i", p.Output.URI, "-frames:v", "1", "-vf", "scale=1:1", "-pix_fmt", "rgb24", "-f", "rawvideo", "-")
			if len(raw) != 3 || raw[0] > 8 || raw[1] > 8 || raw[2] > 8 {
				t.Fatal("replacement was not black", raw)
			}
			pcm := run(ffmpeg, "-v", "error", "-ss", "6.5", "-i", p.Output.URI, "-t", "0.2", "-map", "0:a:0", "-f", "s16le", "-")
			for _, b := range pcm {
				if b != 0 {
					t.Fatal("replacement audio was not silence")
				}
			}
		}
	}
	for _, bad := range []core.TimeRange{{StartSeconds: 0, EndSeconds: 10}, {StartSeconds: 5, EndSeconds: 16}} {
		p := core.Project{SchemaVersion: 1, ID: "edge", Inputs: []core.Source{src}, Output: core.Source{Kind: core.SourceLocalFile, URI: filepath.Join(dir, fmt.Sprintf("edge-%d.mp4", int(bad.StartSeconds)))}, Settings: core.DefaultSettings()}
		p.Settings.Codec, p.Settings.Encoder, p.Settings.Preset = "h264", "cpu", "ultrafast"
		p.Settings.Recovery = &core.Recovery{Mode: "black", SourceFingerprint: media.SourceFingerprint, DamagedRanges: []core.TimeRange{bad}}
		plan, err := compiler.Compile(p, media, caps)
		if err != nil {
			t.Fatal(err)
		}
		plan.JobID = p.Output.URI
		if _, err := runner.Start(ctx, plan, core.Callbacks{}); err != nil {
			t.Fatal("edge replacement", err)
		}
	}
	// A valid container containing corrupted packets must fail final decoding and
	// never become a published recovery, even when the remux itself succeeds.
	output := core.Source{Kind: core.SourceLocalFile, URI: filepath.Join(dir, "must-not-publish.mp4")}
	args := []string{"-hide_banner", "-nostdin", "-n", "-progress", "pipe:1", "-nostats"}
	args = append(args, ffmpegpolicy.LocalInput(src.URI)...)
	args = append(args, "-map", "0:v:0", "-map", "0:a?", "-c", "copy", output.URI)
	_, err = runner.Start(ctx, core.ExecutionPlan{JobID: "verify-reject", Input: src, Output: output, SourceFingerprint: media.SourceFingerprint, DurationSeconds: 16, Args: args, FullDecodeVerification: true}, core.Callbacks{})
	if err == nil {
		t.Fatal("corrupt result passed full decode verification")
	}
	if _, err := os.Stat(output.URI); !os.IsNotExist(err) {
		t.Fatal("failed verification published output")
	}
	// A cancelled scan cannot return a completed report.
	cancelled, stop := context.WithCancel(ctx)
	stop()
	if _, err := runner.ScanHealth(cancelled, media, 5, nil); err == nil {
		t.Fatal("cancelled scan completed")
	}
}
