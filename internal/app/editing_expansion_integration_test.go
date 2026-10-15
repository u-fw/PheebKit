package app

import (
	"context"
	"kompeg/internal/core"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestRealIndependentRangeExportsThroughQueue(t *testing.T) {
	if os.Getenv("KOMPEG_INTEGRATION") != "1" {
		t.Skip("real FFmpeg opt-in")
	}
	t.Setenv("APPDATA", t.TempDir())
	s := NewService()
	defer s.Close()
	dir := t.TempDir()
	input := filepath.Join(dir, "source.mp4")
	environment := discoverEnvironment(binaryConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, environment.FFmpeg.Path, "-hide_banner", "-loglevel", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "testsrc2=size=160x90:rate=10:duration=3", "-f", "lavfi", "-i", "sine=duration=3", "-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac", input)
	hidePlatformProcess(cmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v %s", err, output)
	}
	media, err := s.Probe(input)
	if err != nil {
		t.Fatal(err)
	}
	s.caps = &core.Capabilities{Encoders: []core.Encoder{{Name: "libx264"}, {Name: "aac"}}}
	settings := core.DefaultSettings()
	settings.Codec, settings.Encoder, settings.Preset = "h264", "cpu", "ultrafast"
	projects := []core.Project{}
	for i, start := range []float64{1.5, .2} {
		current := core.CloneSettings(settings)
		current.Trim = &core.TimeRange{StartSeconds: start, EndSeconds: start + 1}
		projects = append(projects, core.Project{SchemaVersion: 1, ID: []string{"part-1", "part-2"}[i], Inputs: []core.Source{media.Source}, Output: source(filepath.Join(dir, []string{"part-01.mp4", "part-02.mp4"}[i])), Settings: current})
	}
	jobs, err := s.StartBatch(projects)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		completed := 0
		for _, job := range jobs {
			current, _ := s.jobs.Get(job.ID)
			if current.State == core.JobFailed {
				t.Fatal(current.Error)
			}
			if current.State == core.JobCompleted {
				if current.Result.Verification == nil || math.Abs(current.Result.Verification.DurationSeconds-1) > .2 {
					t.Fatal("independent export not verified at requested duration")
				}
				completed++
			}
		}
		if completed == 2 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("independent range exports did not finish")
}
