package compiler_test

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"pheebkit/internal/capabilities"
	"pheebkit/internal/compiler"
	"pheebkit/internal/core"
	windowsrunner "pheebkit/internal/runner/windows"
)

func TestIntegrationAudioProcessing(t *testing.T) {
	if os.Getenv("PHEEBKIT_INTEGRATION") != "1" {
		t.Skip("real FFmpeg opt-in")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	ffmpeg, ffprobe := os.Getenv("PHEEBKIT_FFMPEG"), os.Getenv("PHEEBKIT_FFPROBE")
	run := func(combined bool, args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(ctx, ffmpeg, args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		var data []byte
		var err error
		if combined {
			data, err = cmd.CombinedOutput()
		} else {
			data, err = cmd.Output()
		}
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				t.Fatalf("FFmpeg %v %s %s", err, data, e.Stderr)
			}
			t.Fatal(err)
		}
		return data
	}
	dir := t.TempDir()
	src := core.Source{Kind: core.SourceLocalFile, URI: filepath.Join(dir, "audio.mkv")}
	run(true, "-v", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "testsrc2=size=160x90:rate=10:duration=6", "-f", "lavfi", "-i", "sine=frequency=1000:sample_rate=44100:duration=6", "-f", "lavfi", "-i", "sine=frequency=500:sample_rate=44100:duration=6", "-map", "0:v", "-map", "1:a", "-map", "2:a", "-c:v", "libx264", "-preset", "ultrafast", "-c:a", "flac", src.URI)
	runner := windowsrunner.New(ffmpeg, ffprobe)
	media, err := runner.Probe(ctx, src)
	if err != nil {
		t.Fatal(err)
	}
	if media.Streams[1].SampleRate != 44100 || media.Streams[1].Channels != 1 {
		t.Fatal("probe did not expose audio format", media.Streams[1])
	}
	caps := core.Capabilities{Encoders: capabilities.ParseEncoders(string(run(false, "-hide_banner", "-encoders")))}
	entries, err := runner.Catalog(ctx, "filters")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range entries {
		caps.Filters = append(caps.Filters, f.Name)
	}
	count := 0
	execute := func(input core.Source, m core.Media, s core.Settings, ext string) core.Source {
		t.Helper()
		count++
		out := core.Source{Kind: core.SourceLocalFile, URI: filepath.Join(dir, fmt.Sprintf("out-%d.%s", count, ext))}
		p := core.Project{SchemaVersion: 1, ID: fmt.Sprint(count), Inputs: []core.Source{input}, Output: out, Settings: s}
		plan, err := compiler.Compile(p, m, caps)
		if err != nil {
			t.Fatal(err)
		}
		plan.JobID = p.ID
		if _, err := runner.Start(ctx, plan, core.Callbacks{}); err != nil {
			t.Fatal(err)
		}
		return out
	}
	settings := func() core.Settings {
		s := core.DefaultSettings()
		s.Operation = "extract-audio"
		s.Audio = "wav"
		s.AudioSelection = &core.TrackSelection{Mode: "selected", Indices: []int{1}}
		s.Resolution = 0
		s.AudioProcessing = &core.AudioProcessing{NoiseFloorDB: -50, Channels: "source"}
		return s
	}
	pcm := func(out core.Source, start, duration float64, track int) []byte {
		return run(false, "-v", "error", "-ss", fmt.Sprint(start), "-i", out.URI, "-t", fmt.Sprint(duration), "-map", fmt.Sprintf("0:a:%d", track), "-ac", "1", "-ar", "48000", "-f", "f32le", "-")
	}
	rms := func(data []byte) float64 {
		sum := 0.0
		for i := 0; i+3 < len(data); i += 4 {
			v := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[i:])))
			sum += v * v
		}
		if len(data) == 0 {
			t.Fatal("empty PCM")
		}
		return math.Sqrt(sum / float64(len(data)/4))
	}
	base := execute(src, media, settings(), "wav")
	baseRMS := rms(pcm(base, 2, 1, 0))
	eq := settings()
	eq.AudioProcessing.Equalizer = []core.EqualizerBand{{FrequencyHz: 1000, Q: 1, GainDB: -12}}
	filtered := execute(src, media, eq, "wav")
	gain := 20 * math.Log10(rms(pcm(filtered, 2, 1, 0))/baseRMS)
	if math.Abs(gain+12) > .6 {
		t.Fatal("EQ did not attenuate test tone by 12 dB", gain)
	}
	fade := settings()
	fade.AudioProcessing.FadeInSeconds = 1
	fade.AudioProcessing.FadeOutSeconds = 1
	faded := execute(src, media, fade, "wav")
	if rms(pcm(faded, .02, .08, 0)) > baseRMS*.15 || rms(pcm(faded, 5.9, .08, 0)) > baseRMS*.15 || rms(pcm(faded, 2, 1, 0)) < baseRMS*.95 {
		t.Fatal("fade envelope incorrect")
	}
	normalized := settings()
	normalized.AudioProcessing.Normalize = &core.Loudness{IntegratedLUFS: -16, TruePeakDBTP: -1.5, RangeLU: 11}
	norm := execute(src, media, normalized, "wav")
	log := string(run(true, "-hide_banner", "-nostats", "-i", norm.URI, "-af", "loudnorm=print_format=json", "-f", "null", "-"))
	var measurement map[string]string
	if i := strings.LastIndex(log, "{"); i < 0 {
		t.Fatal("no loudness measurement")
	} else if err := json.NewDecoder(strings.NewReader(log[i:])).Decode(&measurement); err != nil {
		t.Fatal(err)
	}
	measured, err := strconv.ParseFloat(measurement["input_i"], 64)
	if err != nil || math.Abs(measured+16) > .8 {
		t.Fatal("normalized loudness outside fixture target", measured, err)
	}
	t.Logf("measured LUFS %.2f; EQ gain %.2f dB", measured, gain)
	for _, item := range []struct {
		mode, ext string
		rate      int
	}{{"aac", "m4a", 44100}, {"opus", "opus", 48000}, {"mp3", "mp3", 24000}, {"flac", "flac", 96000}} {
		s := settings()
		s.Audio = item.mode
		s.AudioProcessing.SampleRate = item.rate
		s.AudioProcessing.Channels = "stereo"
		out := execute(src, media, s, item.ext)
		m, err := runner.Probe(ctx, out)
		if err != nil || m.Streams[0].SampleRate != item.rate || m.Streams[0].Channels != 2 {
			t.Fatal("rate/channel mismatch", item, m, err)
		}
	}
	// Test in a video graph with auto conversion disabled by the depth policy.
	joined := settings()
	joined.Operation = "compress"
	joined.Codec = "h264"
	joined.Encoder = "cpu"
	joined.Preset = "ultrafast"
	joined.Audio = "aac"
	joined.AudioSelection = nil
	joined.Segments = []core.TimeRange{{EndSeconds: 2}, {StartSeconds: 4, EndSeconds: 6}}
	joined.AudioProcessing.FadeInSeconds = 1
	joined.AudioProcessing.FadeOutSeconds = 1
	joined.AudioProcessing.Normalize = &core.Loudness{IntegratedLUFS: -18, TruePeakDBTP: -2, RangeLU: 11}
	joined.AudioProcessing.DenoiseReductionDB = 6
	merged := execute(src, media, joined, "mp4")
	for track := 0; track < 2; track++ {
		middle := rms(pcm(merged, 1.8, .4, track))
		head := rms(pcm(merged, .02, .08, track))
		tail := rms(pcm(merged, 3.9, .08, track))
		if middle < .02 || head > middle*.2 || tail > middle*.2 {
			t.Fatal("fades repeated per segment or audio effects failed", track, head, middle, tail)
		}
	}
	joined.Segments = nil
	joined.Trim = &core.TimeRange{StartSeconds: 1, EndSeconds: 5}
	trimmed := execute(src, media, joined, "mp4")
	if rms(pcm(trimmed, .02, .08, 0)) > rms(pcm(trimmed, 1.5, .5, 0))*.2 {
		t.Fatal("nonzero trim fade used source timestamps")
	}
	noise := core.Source{Kind: core.SourceLocalFile, URI: filepath.Join(dir, "noise.wav")}
	run(true, "-v", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "anoisesrc=color=white:amplitude=0.02:sample_rate=48000:duration=4:seed=41", "-c:a", "pcm_s16le", noise.URI)
	nm, err := runner.Probe(ctx, noise)
	if err != nil {
		t.Fatal(err)
	}
	dn := settings()
	dn.AudioSelection.Indices = []int{0}
	dn.AudioProcessing.DenoiseReductionDB = 20
	dn.AudioProcessing.NoiseFloorDB = -30
	quiet := execute(noise, nm, dn, "wav")
	ratio := rms(pcm(quiet, .5, 3, 0)) / rms(pcm(noise, .5, 3, 0))
	if ratio > .8 {
		t.Fatal("noise fixture not attenuated", ratio)
	}
	t.Logf("seeded noise RMS ratio %.3f", ratio)
}
