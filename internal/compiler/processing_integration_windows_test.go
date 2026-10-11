package compiler_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"kompeg/internal/capabilities"
	"kompeg/internal/compiler"
	"kompeg/internal/core"
	windowsrunner "kompeg/internal/runner/windows"
)

func TestIntegrationMediaOperationsRoundTrip(t *testing.T) {
	if os.Getenv("KOMPEG_INTEGRATION") != "1" {
		t.Skip("set KOMPEG_INTEGRATION=1 to execute installed FFmpeg/FFprobe")
	}
	tool := func(env, name string) string {
		t.Helper()
		if value := os.Getenv(env); value != "" {
			if info, err := os.Stat(value); err == nil && info.IsDir() {
				return filepath.Join(value, name)
			}
			return value
		}
		value, err := exec.LookPath(name)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	ffmpeg, ffprobe := tool("KOMPEG_FFMPEG", "ffmpeg.exe"), tool("KOMPEG_FFPROBE", "ffprobe.exe")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	dir := t.TempDir()
	run := func(tool string, args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(ctx, tool, args...)
		cmd.Dir = dir
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("real FFmpeg command failed: %v\n%s", err, data)
		}
		return data
	}
	stdinRun := func(args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(ctx, ffmpeg, args...)
		cmd.Dir = dir
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		var errOutput bytes.Buffer
		cmd.Stderr = &errOutput
		data, err := cmd.Output()
		if err != nil {
			t.Fatalf("decode failed: %v %s", err, errOutput.String())
		}
		return data
	}
	subPath := filepath.Join(dir, "字幕.srt")
	if err := os.WriteFile(subPath, []byte("1\n00:00:00,500 --> 00:00:01,500\nPheebKit 保留字幕\n"), 0600); err != nil {
		t.Fatal(err)
	}
	input := core.Source{URI: filepath.Join(dir, "多轨 中文.mkv"), Kind: core.SourceLocalFile}
	run(ffmpeg, "-hide_banner", "-loglevel", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "testsrc2=size=128x96:rate=10:duration=3", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=3", "-f", "lavfi", "-i", "sine=frequency=880:sample_rate=48000:duration=3", "-i", subPath, "-map", "0:v:0", "-map", "1:a:0", "-map", "2:a:0", "-map", "3:s:0", "-c:v", "libx264", "-preset", "ultrafast", "-crf", "0", "-c:a:0", "flac", "-c:a:1", "aac", "-c:s", "srt", "-metadata", "title=PheebKit round trip", "-metadata:s:a:0", "language=eng", "-metadata:s:a:1", "language=zho", "-metadata:s:a:1", "title=selected 880 Hz", input.URI)
	runner := windowsrunner.New(ffmpeg, ffprobe)
	media, err := runner.Probe(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(media.Streams) != 4 {
		t.Fatalf("expected true multi-track input: %+v", media)
	}
	caps := core.Capabilities{Encoders: capabilities.ParseEncoders(string(run(ffmpeg, "-hide_banner", "-encoders")))}
	project := core.Project{SchemaVersion: core.ProjectSchemaVersion, ID: "media-operations", Inputs: []core.Source{input}, Output: core.Source{URI: filepath.Join(dir, "selected.mp4"), Kind: core.SourceLocalFile}, Settings: core.DefaultSettings()}
	project.Settings.Codec, project.Settings.Encoder, project.Settings.Preset = "h264", "cpu", "fast"
	project.Settings.AudioSelection = &core.TrackSelection{Mode: "selected", Indices: []int{2}}
	project.Settings.SubtitleSelection = &core.TrackSelection{Mode: "selected", Indices: []int{3}}
	execute := func(p core.Project, m core.Media) core.Media {
		t.Helper()
		plan, err := compiler.Compile(p, m, caps)
		if err != nil {
			t.Fatal(err)
		}
		plan.JobID = filepath.Base(p.Output.URI)
		if _, err := runner.Start(ctx, plan, core.Callbacks{}); err != nil {
			t.Fatal(err)
		}
		out, err := runner.Probe(ctx, p.Output)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}

	t.Run("compression retains selected audio and converts text subtitle", func(t *testing.T) {
		out := execute(project, media)
		if len(out.Streams) != 3 || out.Streams[1].Language != "zho" || out.Streams[2].CodecName != "mov_text" {
			t.Fatalf("selection/metadata/subtitle conversion wrong: %+v", out)
		}
		titles := run(ffprobe, "-v", "error", "-select_streams", "a:0", "-show_entries", "stream_tags=handler_name", "-of", "default=nw=1:nk=1", project.Output.URI)
		if !strings.Contains(string(titles), "selected 880 Hz") {
			t.Fatalf("selected MP4 audio track title lost: %s", titles)
		}
	})
	t.Run("remux copies packet payload and metadata", func(t *testing.T) {
		p := core.CloneProject(project)
		p.Settings.Operation = "remux"
		p.Output.URI = filepath.Join(dir, "copy.mkv")
		out := execute(p, media)
		if len(out.Streams) != 3 || out.Streams[0].CodecName != "h264" || out.Streams[1].CodecName != "aac" || out.Streams[1].Language != "zho" || out.Streams[2].CodecName != "subrip" {
			t.Fatalf("remux streams changed: %+v", out)
		}
		hashes := func(path, stream string) []byte {
			return run(ffprobe, "-v", "error", "-select_streams", stream, "-show_packets", "-show_entries", "packet=data_hash", "-show_data_hash", "sha256", "-of", "csv=p=0", path)
		}
		if !bytes.Equal(hashes(input.URI, "a:1"), hashes(p.Output.URI, "a:0")) || !bytes.Equal(hashes(input.URI, "v:0"), hashes(p.Output.URI, "v:0")) {
			t.Fatal("stream-copy changed encoded packet payload")
		}
		meta := run(ffprobe, "-v", "error", "-show_entries", "format_tags=title", "-of", "default=nw=1:nk=1", p.Output.URI)
		if !strings.Contains(string(meta), "PheebKit round trip") {
			t.Fatalf("global metadata lost: %s", meta)
		}
	})
	t.Run("audio copy produces audio-only output", func(t *testing.T) {
		p := core.CloneProject(project)
		p.Settings.Operation = "extract-audio"
		p.Settings.SubtitleSelection = nil
		p.Settings.Audio = "copy"
		p.Output.URI = filepath.Join(dir, "selected.m4a")
		out := execute(p, media)
		if len(out.Streams) != 1 || out.Streams[0].CodecName != "aac" {
			t.Fatalf("extracted audio has unexpected streams: %+v", out)
		}
	})
	t.Run("encoded audio formats produce matching codecs", func(t *testing.T) {
		for _, format := range []struct{ mode, ext, codec string }{{"aac", ".m4a", "aac"}, {"opus", ".opus", "opus"}, {"mp3", ".mp3", "mp3"}, {"flac", ".flac", "flac"}} {
			p := core.CloneProject(project)
			p.Settings.Operation = "extract-audio"
			p.Settings.SubtitleSelection = nil
			p.Settings.Audio = format.mode
			p.Output.URI = filepath.Join(dir, "encoded"+format.ext)
			out := execute(p, media)
			if len(out.Streams) != 1 || out.Streams[0].CodecName != format.codec {
				t.Fatalf("%s produced wrong codec: %+v", format.mode, out)
			}
		}
	})
	t.Run("WebM text subtitles convert and stream-copy", func(t *testing.T) {
		p := core.CloneProject(project)
		p.Settings.Codec = "av1"
		p.Settings.Audio = "opus"
		p.Settings.Preset = "fast"
		p.Output.URI = filepath.Join(dir, "converted.webm")
		out := execute(p, media)
		if len(out.Streams) != 3 || out.Streams[0].CodecName != "av1" || out.Streams[1].CodecName != "opus" || out.Streams[2].CodecName != "webvtt" {
			t.Fatalf("WebM conversion incompatible: %+v", out)
		}
		copyProject := core.CloneProject(p)
		copyProject.Inputs = []core.Source{p.Output}
		copyProject.Output.URI = filepath.Join(dir, "copied.webm")
		copyProject.Settings.Operation = "remux"
		copyProject.Settings.AudioSelection = &core.TrackSelection{Mode: "selected", Indices: []int{1}}
		copyProject.Settings.SubtitleSelection = &core.TrackSelection{Mode: "selected", Indices: []int{2}}
		copied := execute(copyProject, out)
		if len(copied.Streams) != 3 || copied.Streams[2].CodecName != "webvtt" {
			t.Fatalf("WebM copy lost native subtitle: %+v", copied)
		}
	})
	t.Run("audio encoding trims and applies actual volume", func(t *testing.T) {
		p := core.CloneProject(project)
		p.Settings.Operation = "extract-audio"
		p.Settings.SubtitleSelection = nil
		p.Settings.Audio = "wav"
		p.Settings.VolumeDB = -6
		p.Settings.Trim = &core.TimeRange{StartSeconds: 0.5, EndSeconds: 1.5}
		p.Output.URI = filepath.Join(dir, "quieter.wav")
		out := execute(p, media)
		if len(out.Streams) != 1 || out.Streams[0].CodecName != "pcm_s16le" || math.Abs(out.DurationSeconds-1) > 0.01 {
			t.Fatalf("encoded audio trim failed: %+v", out)
		}
		original := stdinRun("-v", "error", "-ss", "0.5", "-i", input.URI, "-t", "1", "-map", "0:2", "-f", "s16le", "-ac", "1", "-ar", "48000", "pipe:1")
		quieter := stdinRun("-v", "error", "-i", p.Output.URI, "-f", "s16le", "-ac", "1", "-ar", "48000", "pipe:1")
		rms := func(data []byte) float64 {
			sum := 0.0
			for i := 0; i+1 < len(data); i += 2 {
				sample := float64(int16(binary.LittleEndian.Uint16(data[i : i+2])))
				sum += sample * sample
			}
			return math.Sqrt(sum / float64(len(data)/2))
		}
		db := 20 * math.Log10(rms(quieter)/rms(original))
		if math.Abs(db+6) > 0.15 {
			t.Fatalf("requested -6dB, measured %.3fdB", db)
		}
	})
	t.Run("extract each editable text format", func(t *testing.T) {
		for _, ext := range []string{".srt", ".ass", ".vtt"} {
			p := core.CloneProject(project)
			p.Settings.Operation = "extract-subtitle"
			p.Output.URI = filepath.Join(dir, "text"+ext)
			execute(p, media)
			data, err := os.ReadFile(p.Output.URI)
			if err != nil || !bytes.Contains(data, []byte("PheebKit 保留字幕")) {
				t.Fatalf("subtitle content lost in %s: %s %v", ext, data, err)
			}
		}
	})
	t.Run("crop rotate and flip change decoded pixels", func(t *testing.T) {
		p := core.CloneProject(project)
		p.Settings.Quality = 0
		p.Settings.Audio = "none"
		p.Settings.SubtitleSelection = nil
		p.Settings.Transform = &core.VideoTransform{Crop: &core.CropMargins{Top: 8, Bottom: 8, Left: 8, Right: 8}, Rotation: 90, FlipHorizontal: true}
		p.Output.URI = filepath.Join(dir, "transformed.mp4")
		out := execute(p, media)
		if len(out.Streams) != 1 || out.Streams[0].Width != 80 || out.Streams[0].Height != 112 {
			t.Fatalf("crop/rotate geometry wrong: %+v", out)
		}
		// Lossless source/output permit a pixel comparison after independently
		// asking FFmpeg for the specified transform at the first source frame.
		expected := stdinRun("-v", "error", "-noautorotate", "-i", input.URI, "-vf", "crop=112:80:8:8:exact=1,transpose=clock,hflip", "-frames:v", "1", "-pix_fmt", "yuv420p", "-f", "rawvideo", "pipe:1")
		actual := stdinRun("-v", "error", "-i", p.Output.URI, "-frames:v", "1", "-pix_fmt", "yuv420p", "-f", "rawvideo", "pipe:1")
		if !bytes.Equal(expected, actual) {
			t.Fatal("encoded transform did not match the specified pixels")
		}
	})
	t.Run("display rotation metadata does not apply transform twice", func(t *testing.T) {
		rotated := core.Source{URI: filepath.Join(dir, "rotation-metadata.mp4"), Kind: core.SourceLocalFile}
		run(ffmpeg, "-v", "error", "-nostdin", "-n", "-display_rotation:v:0", "90", "-i", input.URI, "-map", "0:v:0", "-c:v", "copy", rotated.URI)
		rotatedMedia, err := runner.Probe(ctx, rotated)
		if err != nil {
			t.Fatal(err)
		}
		p := core.CloneProject(project)
		p.Inputs = []core.Source{rotated}
		p.Settings.Audio = "none"
		p.Settings.AudioSelection = nil
		p.Settings.SubtitleSelection = nil
		p.Settings.Transform = &core.VideoTransform{Rotation: 90}
		p.Settings.Transform.Crop = &core.CropMargins{Top: 4, Bottom: 8, Left: 6, Right: 10}
		p.Settings.Quality = 0
		p.Output.URI = filepath.Join(dir, "rotation-output.mp4")
		out := execute(p, rotatedMedia)
		if rotatedMedia.Streams[0].DisplayRotation != 90 && rotatedMedia.Streams[0].DisplayRotation != 270 {
			t.Fatalf("display rotation metadata was not parsed: %+v", rotatedMedia.Streams[0])
		}
		if out.Streams[0].Width != 116 || out.Streams[0].Height != 80 {
			t.Fatalf("unexpected encoded rotation dimensions: %+v", out)
		}
		// First decode the same auto-oriented source frame that preview shows.
		// Then independently crop and rotate its three YUV planes in Go.
		displayed := stdinRun("-v", "error", "-i", rotated.URI, "-frames:v", "1", "-pix_fmt", "yuv420p", "-f", "rawvideo", "pipe:1")
		expected := cropRotate420(displayed, 96, 128, 6, 10, 4, 8)
		actual := stdinRun("-v", "error", "-i", p.Output.URI, "-frames:v", "1", "-pix_fmt", "yuv420p", "-f", "rawvideo", "pipe:1")
		if !bytes.Equal(expected, actual) {
			t.Fatal("output crop/rotation did not match auto-oriented preview pixels")
		}
		data := run(ffprobe, "-v", "error", "-show_entries", "stream_side_data=rotation", "-of", "json", p.Output.URI)
		var probe struct {
			Streams []struct {
				Side []struct {
					Rotation int `json:"rotation"`
				} `json:"side_data_list"`
			} `json:"streams"`
		}
		if err := json.Unmarshal(data, &probe); err != nil {
			t.Fatal(err)
		}
		for _, s := range probe.Streams {
			for _, side := range s.Side {
				if side.Rotation != 0 {
					t.Fatalf("already rotated pixels retained rotation matrix: %s", data)
				}
			}
		}
	})
	for _, entry := range mustReadDir(t, dir) {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), ".kompeg-") {
			t.Fatalf("owned execution scratch leaked: %s", entry.Name())
		}
	}
	t.Logf("Real FFmpeg selected tracks, packet-copy, extraction, text formats, volume and transforms passed: %s", fmt.Sprintf("%d streams", len(media.Streams)))
}

func cropRotate420(input []byte, width, height, left, right, top, bottom int) []byte {
	var output []byte
	offset := 0
	for plane := 0; plane < 3; plane++ {
		w, h, l, r, up, down := width, height, left, right, top, bottom
		if plane > 0 {
			w /= 2
			h /= 2
			l /= 2
			r /= 2
			up /= 2
			down /= 2
		}
		cropWidth, cropHeight := w-l-r, h-up-down
		rotated := make([]byte, cropWidth*cropHeight)
		for y := 0; y < cropHeight; y++ {
			for x := 0; x < cropWidth; x++ {
				rotated[x*cropHeight+cropHeight-1-y] = input[offset+(up+y)*w+l+x]
			}
		}
		output = append(output, rotated...)
		offset += w * h
	}
	return output
}

func mustReadDir(t *testing.T, path string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}
