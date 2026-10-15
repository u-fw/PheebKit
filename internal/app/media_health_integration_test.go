package app

import (
	"context"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"kompeg/internal/core"
)

func TestRealHealthAndRecoveryQueue(t *testing.T) {
	if os.Getenv("KOMPEG_INTEGRATION") != "1" {
		t.Skip("real FFmpeg opt-in")
	}
	t.Setenv("APPDATA", t.TempDir())
	s := NewService()
	defer s.Close()
	dir := t.TempDir()
	input := filepath.Join(dir, "source.mp4")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	env := discoverEnvironment(binaryConfig{})
	cmd := exec.CommandContext(ctx, env.FFmpeg.Path, "-v", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "testsrc2=size=160x90:rate=10:duration=3", "-c:v", "libx264", "-preset", "ultrafast", input)
	hidePlatformProcess(cmd)
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, data)
	}
	m, err := s.Probe(input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.StartMediaHealthScan(input, 5); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for s.healthBusy.Load() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if report := s.GetMediaHealthScan(); report.State != "completed" || len(report.DamagedRanges) != 0 {
		t.Fatal("healthy service scan failed", report)
	}
	s.caps = &core.Capabilities{Encoders: []core.Encoder{{Name: "libx264"}}, Filters: []string{"concat", "trim", "setpts"}}
	p := core.Project{SchemaVersion: 1, ID: "recovery-queue", Inputs: []core.Source{m.Source}, Output: source(filepath.Join(dir, "recovered.mp4")), Settings: core.DefaultSettings()}
	p.Settings.Codec, p.Settings.Encoder, p.Settings.Preset, p.Settings.Audio = "h264", "cpu", "ultrafast", "none"
	p.Settings.Recovery = &core.Recovery{Mode: "delete", SourceFingerprint: m.SourceFingerprint, DamagedRanges: []core.TimeRange{{StartSeconds: 1, EndSeconds: 2}}}
	job, err := s.Start(p)
	if err != nil {
		t.Fatal(err)
	}
	p.Settings.Recovery.DamagedRanges[0].EndSeconds = 99
	deadline = time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		current, _ := s.jobs.Get(job.ID)
		if current.State == core.JobFailed {
			t.Fatal(current.Error)
		}
		if current.State == core.JobCompleted {
			if current.Result.Verification == nil || !current.Result.Verification.FullDecodePassed || math.Abs(current.Result.Verification.DurationSeconds-2) > .2 {
				t.Fatal("missing full queue verification")
			}
			if current.Project.Settings.Recovery.DamagedRanges[0].EndSeconds != 2 {
				t.Fatal("queue recovery snapshot changed")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("recovery queue timed out")
}
