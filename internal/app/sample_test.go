package app

import (
	"context"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kompeg/internal/core"
	"kompeg/internal/sourcefile"
)

func TestSampleUsesDistinctBoundedRangeAndProportionalBudget(t *testing.T) {
	s, projects := batchService(t)
	p := projects[0]
	media := s.media[p.Inputs[0].URI]
	media.Streams = append(media.Streams, core.Stream{Index: 1, CodecType: "audio", CodecName: "aac"})
	s.media[p.Inputs[0].URI] = media
	p.Settings.Audio = "copy"
	p.Settings.Mode = "size"
	p.Settings.TargetSizeMB = 100
	p.Settings.Trim = &core.TimeRange{StartSeconds: 2, EndSeconds: 8}
	job, err := s.StartSample(p, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(job.ProjectID, "sample-") || job.Trim.StartSeconds != 2 || job.Trim.EndSeconds != 4 || job.Output.URI == p.Output.URI || job.Project.Settings.Audio != "aac" || math.Abs(job.Project.Settings.TargetSizeMB-100.0/3) > 1e-8 || job.SourceFingerprint == "" {
		t.Fatalf("bad sample: %+v", job)
	}
	if p.Settings.Audio != "copy" || p.Settings.Trim.EndSeconds != 8 || p.Settings.TargetSizeMB != 100 {
		t.Fatal("sample mutated the full task")
	}
	for _, seconds := range []float64{0, 31, math.NaN(), math.Inf(1)} {
		if _, err := s.StartSample(p, seconds); err == nil {
			t.Fatal("invalid sample length accepted")
		}
	}
	p.Settings.Operation = "remux"
	if _, err := s.StartSample(p, 2); err == nil {
		t.Fatal("utility job sampled as compression")
	}
	if _, err := s.GetSampleComparison(job.ID); err == nil {
		t.Fatal("unfinished sample comparison accepted")
	}
}

func TestSampleUsesFirstOrderedSegmentAndFullBudget(t *testing.T) {
	s, projects := batchService(t)
	p := projects[0]
	p.Settings.Segments = []core.TimeRange{{StartSeconds: 5, EndSeconds: 8}, {StartSeconds: 0, EndSeconds: 1}}
	p.Settings.Mode = "size"
	p.Settings.TargetSizeMB = 100
	job, err := s.StartSample(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	if job.Trim.StartSeconds != 5 || job.Trim.EndSeconds != 6 || len(job.Project.Settings.Segments) != 0 || job.Project.Settings.TargetSizeMB != 25 {
		t.Fatal("sample lost first-range or budget policy")
	}
	if len(p.Settings.Segments) != 2 {
		t.Fatal("sample changed editor ranges")
	}
}

func TestRealSampleJobAndComparisonRejectChangedSource(t *testing.T) {
	if os.Getenv("KOMPEG_INTEGRATION") != "1" {
		t.Skip("set KOMPEG_INTEGRATION=1")
	}
	t.Setenv("APPDATA", t.TempDir())
	s := NewService()
	defer s.Close()
	dir := t.TempDir()
	input := filepath.Join(dir, "sample-source.mp4")
	environment := discoverEnvironment(binaryConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, environment.FFmpeg.Path, "-hide_banner", "-loglevel", "error", "-nostdin", "-f", "lavfi", "-i", "testsrc2=size=160x90:rate=10", "-f", "lavfi", "-i", "sine=frequency=880:sample_rate=48000", "-t", "3", "-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac", input)
	hidePlatformProcess(cmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v %s", err, output)
	}
	media, err := s.Probe(input)
	if err != nil {
		t.Fatal(err)
	}
	s.caps = &core.Capabilities{Encoders: []core.Encoder{{Name: "libx264", MediaType: "video"}, {Name: "aac", MediaType: "audio"}}}
	settings := core.DefaultSettings()
	settings.Codec, settings.Encoder, settings.Preset, settings.Audio = "h264", "cpu", "ultrafast", "copy"
	p := core.Project{SchemaVersion: 1, ID: "full", Inputs: []core.Source{media.Source}, Output: source(filepath.Join(dir, "full.mp4")), Settings: settings}
	job, err := s.StartSample(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		current, _ := s.jobs.Get(job.ID)
		if current.State == core.JobCompleted {
			job = current
			break
		}
		if current.State == core.JobFailed {
			t.Fatal(current.Error)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if job.State != core.JobCompleted {
		t.Fatal("sample did not finish")
	}
	comparison, err := s.GetSampleComparison(job.ID)
	if err != nil || math.Abs(comparison.Output.DurationSeconds-1) > 0.15 {
		t.Fatal("comparison result invalid", err)
	}
	for _, uri := range []string{comparison.Original.Source.URI, comparison.Output.Source.URI} {
		if _, err := s.GetVideoFrame(uri, 0.5); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(p.Output.URI); !os.IsNotExist(err) {
		t.Fatal("sample wrote full target")
	}
	if err := os.WriteFile(input, []byte("replaced source"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetSampleComparison(job.ID); !errors.Is(err, sourcefile.ErrChanged) {
		t.Fatalf("comparison accepted replaced source: %v", err)
	}
}
