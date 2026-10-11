package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kompeg/internal/core"
	"kompeg/internal/jobs"
	"kompeg/internal/sourcefile"
)

func batchService(t *testing.T) (*Service, []core.Project) {
	t.Helper()
	t.Setenv("APPDATA", t.TempDir())
	runner := &lifecycleRunner{started: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{ctx: ctx, cancel: cancel, runner: &runnerProxy{current: newPlatformRunner(Environment{})}, jobs: jobs.New(runner, nil), media: make(map[string]core.Media),
		caps: &core.Capabilities{Encoders: []core.Encoder{{Name: "libx264", MediaType: "video"}, {Name: "aac", MediaType: "audio"}}}}
	s.jobs.SetQueuePaused(true)
	t.Cleanup(s.Close)
	dir := t.TempDir()
	projects := make([]core.Project, 2)
	for i := range projects {
		input := filepath.Join(dir, fmt.Sprintf("input-%d.mp4", i))
		if err := os.WriteFile(input, []byte("fixture identity"), 0600); err != nil {
			t.Fatal(err)
		}
		fingerprint, err := sourcefile.Fingerprint(input)
		if err != nil {
			t.Fatal(err)
		}
		s.media[input] = core.Media{Source: source(input), SourceFingerprint: fingerprint, DurationSeconds: 10, Streams: []core.Stream{{Index: 0, CodecType: "video", CodecName: "h264", Width: 160, Height: 90, FrameRate: 10}}}
		settings := core.DefaultSettings()
		settings.Codec, settings.Encoder = "h264", "cpu"
		projects[i] = core.Project{SchemaVersion: core.ProjectSchemaVersion, ID: fmt.Sprintf("project-%d", i), Inputs: []core.Source{source(input)}, Output: source(filepath.Join(dir, fmt.Sprintf("output-%d.mp4", i))), Settings: settings}
	}
	return s, projects
}

func TestStartBatchCompilesAllBeforeQueuingAndHonorsExit(t *testing.T) {
	s, projects := batchService(t)
	broken := append([]core.Project(nil), projects...)
	broken[1].Settings.Codec = "unsupported"
	if _, err := s.StartBatch(broken); err == nil || !strings.Contains(err.Error(), "第 2 个文件") {
		t.Fatal("invalid second project accepted or missing file context", err)
	}
	if len(s.jobs.List()) != 0 {
		t.Fatal("compilation failure queued the valid first project")
	}
	duplicate := append([]core.Project(nil), projects...)
	duplicate[1].Output = duplicate[0].Output
	if _, err := s.StartBatch(duplicate); err == nil {
		t.Fatal("batch output collision accepted")
	}
	if len(s.jobs.List()) != 0 {
		t.Fatal("admission failure queued a partial batch")
	}
	queued, err := s.StartBatch(projects)
	if err != nil || len(queued) != 2 || len(s.QueueStatus().PendingIDs) != 2 {
		t.Fatalf("valid batch: %+v %v", queued, err)
	}
	if s.CanClose() {
		t.Fatal("paused waiting batch bypassed close protection")
	}
	s.lifecycleMu.Lock()
	s.exiting = true
	s.lifecycleMu.Unlock()
	if _, err := s.StartBatch(projects); err == nil {
		t.Fatal("batch accepted while exiting")
	}
	if len(s.jobs.List()) != 2 {
		t.Fatal("exit rejection changed the queue")
	}
}

func TestBatchRequestBoundsAndUnavailableNativeDialogs(t *testing.T) {
	s, _ := batchService(t)
	for _, projects := range [][]core.Project{nil, make([]core.Project, 33)} {
		if _, err := s.StartBatch(projects); err == nil {
			t.Fatal("invalid batch count accepted")
		}
	}
	if _, err := s.ChooseMediaPaths(); err == nil {
		t.Fatal("native multi-picker silently succeeded without an app")
	}
	if _, err := s.ChooseOutputDirectory("C:\\media"); err == nil {
		t.Fatal("native directory picker silently succeeded without an app")
	}
}
