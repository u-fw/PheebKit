package windows

import (
	"context"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"kompeg/internal/compiler"
	"kompeg/internal/core"
)

func TestIntegrationTrimExactNonKeyframeRange(t *testing.T) {
	if os.Getenv("KOMPEG_INTEGRATION") != "1" {
		t.Skip("set KOMPEG_INTEGRATION=1 for installed FFmpeg")
	}
	r := New(os.Getenv("KOMPEG_FFMPEG"), os.Getenv("KOMPEG_FFPROBE"))
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	dir := t.TempDir()
	input := core.Source{URI: filepath.Join(dir, "trim source.mkv"), Kind: core.SourceLocalFile}
	// Only the middle two seconds are blue. A six-second GOP tests accurate
	// input seeking beyond the preceding keyframe, rather than just duration.
	generate := exec.CommandContext(ctx, r.FFmpeg, "-hide_banner", "-loglevel", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "color=red:s=320x180:r=30:d=2[r];color=blue:s=320x180:r=30:d=2[b];color=green:s=320x180:r=30:d=2[g];[r][b][g]concat=n=3:v=1:a=0", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=6", "-c:v", "libx264", "-g", "180", "-sc_threshold", "0", "-preset", "fast", "-c:a", "flac", input.URI)
	hideProcess(generate)
	if data, err := generate.CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v\n%s", err, data)
	}
	m, err := r.Probe(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	caps, err := r.Capabilities(ctx)
	if err != nil {
		t.Fatal(err)
	}
	p := core.Project{SchemaVersion: 1, ID: "trim", Inputs: []core.Source{input}, Output: core.Source{URI: filepath.Join(dir, "trim output.mp4"), Kind: core.SourceLocalFile}, Settings: core.DefaultSettings()}
	p.Settings.Codec, p.Settings.Encoder = "h264", "cpu"
	p.Settings.Trim = &core.TimeRange{StartSeconds: 2.5, EndSeconds: 3.75}
	plan, err := compiler.Compile(p, m, caps)
	if err != nil {
		t.Fatal(err)
	}
	plan.JobID = "trim-integration"
	updates := 0
	_, err = r.Start(ctx, plan, core.Callbacks{Progress: func(progress core.Progress) {
		updates++
		if progress.TotalTimeSeconds != 1.25 || progress.Percent < 0 || progress.Percent > 100 {
			t.Errorf("wrong segment progress: %+v", progress)
		}
	}})
	if err != nil {
		t.Fatal(err)
	}
	out, err := r.Probe(ctx, p.Output)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out.DurationSeconds-1.25) > 0.1 || len(out.Streams) != 2 || out.Streams[0].CodecName != "h264" || out.Streams[1].CodecName != "aac" || updates == 0 {
		t.Fatalf("invalid trimmed output: %+v, progress updates=%d", out, updates)
	}
	decode := exec.CommandContext(ctx, r.FFmpeg, "-hide_banner", "-loglevel", "error", "-i", p.Output.URI, "-frames:v", "1", "-vf", "crop=2:2:100:100,format=rgb24", "-f", "rawvideo", "pipe:1")
	hideProcess(decode)
	pixels, err := decode.Output()
	if err != nil || len(pixels) != 12 || pixels[0] > 15 || pixels[1] > 15 || pixels[2] < 230 {
		t.Fatalf("wrong starting frame, expected blue: %v %v", pixels, err)
	}
	t.Logf("verified non-keyframe 2.5–3.75 second slice: %.3fs, blue starting frame, H.264/AAC, %d progress updates", out.DurationSeconds, updates)
}
