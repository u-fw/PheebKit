package windows

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"pheebkit/internal/compiler"
	"pheebkit/internal/core"
)

func TestIntegrationGPUDetectionAndQualifiedEncodeModes(t *testing.T) {
	if os.Getenv("PHEEBKIT_INTEGRATION") != "1" {
		t.Skip("set PHEEBKIT_INTEGRATION=1 to execute GPU smoke trials and installed FFmpeg")
	}
	runner := New(os.Getenv("PHEEBKIT_FFMPEG"), os.Getenv("PHEEBKIT_FFPROBE"))
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	caps, err := runner.Capabilities(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, gpu := range caps.GPUs {
		t.Logf("native adapter: %s (%s)", gpu.Name, gpu.Vendor)
	}
	if report := os.Getenv("PHEEBKIT_GPU_REPORT"); report != "" {
		data, err := json.MarshalIndent(caps, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(report, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	input := core.Source{URI: filepath.Join(dir, "gpu input.mp4"), Kind: core.SourceLocalFile}
	generate := exec.CommandContext(ctx, runner.FFmpeg, "-hide_banner", "-loglevel", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "testsrc2=size=640x360:rate=30:duration=2", "-c:v", "libx264", "-preset", "ultrafast", "-crf", "18", input.URI)
	hideProcess(generate)
	if output, err := generate.CombinedOutput(); err != nil {
		t.Fatalf("generate GPU fixture: %v\n%s", err, output)
	}
	media, err := runner.Probe(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	input10 := core.Source{URI: filepath.Join(dir, "gpu 10-bit input.mp4"), Kind: core.SourceLocalFile}
	generate10 := exec.CommandContext(ctx, runner.FFmpeg, "-hide_banner", "-loglevel", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "testsrc2=size=640x360:rate=30:duration=2", "-c:v", "libx265", "-preset", "ultrafast", "-x265-params", "log-level=error", "-crf", "18", "-pix_fmt", "yuv420p10le", input10.URI)
	hideProcess(generate10)
	if output, err := generate10.CombinedOutput(); err != nil {
		t.Fatalf("generate genuine 10-bit GPU fixture: %v\n%s", err, output)
	}
	media10, err := runner.Probe(ctx, input10)
	if err != nil || len(media10.Streams) != 1 || media10.Streams[0].BitDepth != 10 {
		t.Fatalf("fixture did not probe as genuine 10-bit media: %+v %v", media10, err)
	}
	verified := 0
	for _, encoder := range caps.VideoEncoders {
		if encoder.Kind != "gpu" {
			continue
		}
		t.Logf("GPU encoder %s available=%t formats=%v reason=%s", encoder.Name, encoder.Available, encoder.PixelFormats, encoder.Reason)
		if !encoder.Available {
			continue
		}
		for _, pixelFormat := range encoder.PixelFormats {
			for _, mode := range []string{"quality", "bitrate", "size"} {
				name := fmt.Sprintf("%s_%s_%s", encoder.Name, pixelFormat, mode)
				t.Run(name, func(t *testing.T) {
					intent := media
					expectedDepth := 8
					if pixelFormat == "p010le" {
						intent, expectedDepth = media10, 10
					}
					project := core.Project{SchemaVersion: 1, ID: name, Inputs: []core.Source{intent.Source},
						Output: core.Source{URI: filepath.Join(dir, name+".mp4"), Kind: core.SourceLocalFile}, Settings: core.DefaultSettings()}
					project.Settings.Codec, project.Settings.Encoder = encoder.Codec, encoder.Name
					project.Settings.Mode, project.Settings.BitrateKbps, project.Settings.TargetSizeMB = mode, 2000, 1
					plan, err := compiler.Compile(project, intent, caps)
					if err != nil {
						t.Fatal(err)
					}
					plan.JobID = name
					result, err := runner.Start(ctx, plan, core.Callbacks{})
					if err != nil {
						t.Fatal(err)
					}
					output, err := runner.Probe(ctx, project.Output)
					if err != nil {
						t.Fatal(err)
					}
					if len(output.Streams) != 1 || output.Streams[0].CodecName != encoder.Codec || output.Streams[0].BitDepth != expectedDepth || result.OutputSizeBytes <= 0 || math.Abs(output.DurationSeconds-intent.DurationSeconds) > 0.15 {
						t.Fatalf("invalid GPU output: %+v result=%+v", output, result)
					}
					t.Logf("%s encoded %d bytes in %.3fs; %.2fx; depth=%d", plan.Encoder, result.OutputSizeBytes, result.ElapsedSeconds, result.AverageSpeed, expectedDepth)
					verified++
				})
			}
		}
	}
	if verified == 0 {
		t.Log("No GPU passed its runtime trial; automatic selection uses the detected CPU encoder.")
	}
	// The automatic default must agree with the actual qualified device result.
	selected := ""
	for _, encoder := range caps.VideoEncoders {
		if encoder.Codec == "hevc" && encoder.Available {
			selected = encoder.Name
			break
		}
	}
	if selected != "" {
		project := core.Project{SchemaVersion: 1, Inputs: []core.Source{input}, Output: core.Source{URI: filepath.Join(dir, "auto.mp4"), Kind: core.SourceLocalFile}, Settings: core.DefaultSettings()}
		plan, err := compiler.Compile(project, media, caps)
		if err != nil || plan.Encoder != selected {
			t.Fatalf("auto default disagrees with qualified hardware: %+v %v; want=%s", plan, err, selected)
		}
		t.Logf("automatic HEVC selects %s (%s)", plan.Encoder, plan.EncoderKind)
	}
}
