package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pheebkit/internal/core"
	"pheebkit/internal/localstore"
)

func TestRealRecoveredDraftReprobesAndProcessesOnlyAfterExplicitStart(t *testing.T) {
	if os.Getenv("PHEEBKIT_INTEGRATION") != "1" {
		t.Skip("set PHEEBKIT_INTEGRATION=1")
	}
	t.Setenv("APPDATA", t.TempDir())
	s := NewService()
	defer s.Close()
	dir := t.TempDir()
	input := filepath.Join(dir, "recover-source.mp4")
	environment := discoverEnvironment(binaryConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, environment.FFmpeg.Path, "-hide_banner", "-loglevel", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "testsrc2=size=160x90:rate=10:duration=2", "-f", "lavfi", "-i", "sine=duration=2", "-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac", input)
	hidePlatformProcess(cmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate recovery fixture: %v %s", err, output)
	}
	settings := core.DefaultSettings()
	settings.Codec, settings.Encoder, settings.Preset, settings.AudioBitrateKbps = "h264", "cpu", "ultrafast", 64
	p := core.Project{SchemaVersion: 1, ID: "interrupted-editor", Inputs: []core.Source{source(input)}, Output: source(filepath.Join(dir, "recovered.mp4")), Settings: settings}
	if err := s.SaveRecoveryDraft(p, 1); err != nil {
		t.Fatal(err)
	}
	data, path, err := readEditor()
	if err != nil {
		t.Fatal(err)
	}
	// Represent a prior process that exited before its latest checkpoint was saved as a project.
	if err := localstore.Update(path, &data, func() error { data.Drafts[0].PID = 0; return nil }); err != nil {
		t.Fatal(err)
	}
	reopened := NewService()
	defer reopened.Close()
	reopened.caps = &core.Capabilities{Encoders: []core.Encoder{{Name: "libx264", MediaType: "video"}, {Name: "aac", MediaType: "audio"}}}
	recovered, err := reopened.RestoreRecoveryDraft(data.Drafts[0].ID)
	if err != nil || !recovered.NeedsSave || recovered.Media.SourceFingerprint == "" || len(recovered.Media.Streams) != 2 || len(reopened.jobs.List()) != 0 {
		t.Fatal("real recovery did not safely re-probe", err)
	}
	plan, err := reopened.Preview(recovered.Project)
	if err != nil || !strings.Contains(strings.Join(plan.Args, " "), "-b:a 64k") {
		t.Fatal("custom audio intent lost during recovery", err)
	}
	job, err := reopened.Start(recovered.Project)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		current, _ := reopened.jobs.Get(job.ID)
		if current.State == core.JobCompleted {
			if current.Result.Verification == nil || current.Result.Verification.AudioStreams != 1 {
				t.Fatal("recovered output was not verified")
			}
			return
		}
		if current.State == core.JobFailed {
			t.Fatal(current.Error)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("recovered task did not finish")
}
