package app

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pheebkit/internal/core"
	"pheebkit/internal/jobs"
	"pheebkit/internal/runner/windows"
	"pheebkit/internal/sourcefile"
)

type probeMutationRunner struct{ mutate func() }

func (r *probeMutationRunner) Probe(_ context.Context, source core.Source) (core.Media, error) {
	r.mutate()
	return core.Media{Source: source, DurationSeconds: 1}, nil
}
func (*probeMutationRunner) Capabilities(context.Context) (core.Capabilities, error) {
	return core.Capabilities{}, nil
}
func (*probeMutationRunner) Start(context.Context, core.ExecutionPlan, core.Callbacks) (core.Result, error) {
	return core.Result{}, core.ErrUnsupported
}
func (*probeMutationRunner) Cancel(string) error { return nil }

func TestProbeRejectsMutationBeforeCaching(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input.mp4")
	if err := os.WriteFile(path, []byte("original media"), 0600); err != nil {
		t.Fatal(err)
	}
	service := NewService()
	defer service.Close()
	service.runner.replace(&probeMutationRunner{mutate: func() {
		if err := os.WriteFile(path, []byte("modified media"), 0600); err != nil {
			t.Fatal(err)
		}
	}})
	if _, err := service.Probe(path); !errors.Is(err, sourcefile.ErrChanged) {
		t.Fatalf("probe published changed media: %v", err)
	}
	if len(service.media) != 0 {
		t.Fatal("failed probe stored stale metadata")
	}
}

func TestIntegrationReplacementNeedsExplicitReimport(t *testing.T) {
	if os.Getenv("PHEEBKIT_INTEGRATION") != "1" {
		t.Skip("set PHEEBKIT_INTEGRATION=1 for real media replacement regression")
	}
	directory := t.TempDir()
	input, replacement := filepath.Join(directory, "源视频.mp4"), filepath.Join(directory, "replacement.mp4")
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	environment := discoverEnvironment(binaryConfig{})
	for index, path := range []string{input, replacement} {
		args := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-f", "lavfi", "-i", "color=c=blue:s=160x90:r=10"}
		if index == 1 {
			args = append(args, "-f", "lavfi", "-i", "sine=frequency=880:sample_rate=48000")
		}
		args = append(args, "-t", []string{"1", "2"}[index], "-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac", path)
		command := exec.CommandContext(ctx, environment.FFmpeg.Path, args...)
		hidePlatformProcess(command)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("generate media: %v %s", err, output)
		}
	}
	oldBytes, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	newBytes, err := os.ReadFile(replacement)
	if err != nil {
		t.Fatal(err)
	}
	// Valid MP4 files tolerate trailing padding; size alone cannot distinguish them.
	size := max(len(oldBytes), len(newBytes))
	oldBytes, newBytes = append(oldBytes, make([]byte, size-len(oldBytes))...), append(newBytes, make([]byte, size-len(newBytes))...)
	if err := os.WriteFile(input, oldBytes, 0600); err != nil {
		t.Fatal(err)
	}
	service := NewService()
	defer service.Close()
	media, err := service.Probe(input)
	if err != nil {
		t.Fatal(err)
	}
	service.caps = &core.Capabilities{Encoders: []core.Encoder{{Name: "libx264", MediaType: "video"}, {Name: "aac", MediaType: "audio"}}}
	project := core.Project{SchemaVersion: 1, ID: "replacement", Inputs: []core.Source{source(input)}, Output: source(filepath.Join(directory, "output.mp4")), Settings: core.DefaultSettings()}
	project.Settings.Codec, project.Settings.Encoder = "h264", "cpu"
	initial, err := service.Preview(project)
	if err != nil {
		t.Fatal(err)
	}
	if initial.SourceFingerprint == "" || initial.SourceFingerprint != media.SourceFingerprint {
		t.Fatal("plan lost its probed source identity")
	}
	stat, err := os.Stat(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, newBytes, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(input, stat.ModTime(), stat.ModTime()); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := service.Preview(project); !errors.Is(err, sourcefile.ErrChanged) {
			t.Fatalf("preview silently reprobed replacement (attempt %d): %v", attempt, err)
		}
	}
	if _, err := service.Start(project); !errors.Is(err, sourcefile.ErrChanged) {
		t.Fatalf("Start used stale media: %v", err)
	}
	if _, err := service.GetThumbnail(input); !errors.Is(err, sourcefile.ErrChanged) {
		t.Fatalf("thumbnail used stale stream metadata: %v", err)
	}
	updated, err := service.Probe(input)
	if err != nil {
		t.Fatal(err)
	}
	if updated.SizeBytes != media.SizeBytes || updated.DurationSeconds < 1.9 || len(updated.Streams) != 2 || updated.SourceFingerprint == media.SourceFingerprint {
		t.Fatalf("reimport did not refresh equal-size media: old=%+v updated=%+v", media, updated)
	}
	plan, err := service.Preview(project)
	if err != nil || plan.DurationSeconds < 1.9 || plan.SourceFingerprint != updated.SourceFingerprint {
		t.Fatalf("reimport did not repair preview: %+v %v", plan, err)
	}
	// Validate queued execution independently of Service's immediate preview.
	t.Run("queued source replacement is rejected before FFmpeg starts", func(t *testing.T) {
		gate := &queuedIdentityRunner{runner: windows.New(filepath.Join(directory, "must-not-run.exe"), environment.FFprobe.Path), started: make(chan struct{}), release: make(chan struct{})}
		manager := jobs.New(gate, nil)
		defer manager.Close()
		if _, err := manager.Enqueue(core.ExecutionPlan{JobID: "blocker"}); err != nil {
			t.Fatal(err)
		}
		select {
		case <-gate.started:
		case <-time.After(2 * time.Second):
			t.Fatal("blocker did not start")
		}
		plan.JobID = "replaced-while-queued"
		queued, err := manager.Enqueue(plan)
		if err != nil || queued.State != core.JobQueued {
			t.Fatalf("could not queue plan: %+v %v", queued, err)
		}
		if err := os.WriteFile(input, oldBytes, 0600); err != nil {
			t.Fatal(err)
		}
		close(gate.release)
		deadline := time.After(3 * time.Second)
		for {
			job, _ := manager.Get(plan.JobID)
			if job.State == core.JobFailed {
				if !strings.Contains(job.Error, "重新导入") {
					t.Fatalf("wrong queued failure: %+v", job)
				}
				break
			}
			select {
			case <-deadline:
				t.Fatalf("queued replacement not rejected: %+v", job)
			case <-time.After(10 * time.Millisecond):
			}
		}
		if _, err := os.Stat(plan.Output.URI); !os.IsNotExist(err) {
			t.Fatalf("changed source created output: %v", err)
		}
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".pheebkit-") {
				t.Fatal("preflight rejection created a scratch directory")
			}
		}
	})
}

type queuedIdentityRunner struct {
	runner           *windows.Runner
	started, release chan struct{}
}

func (r *queuedIdentityRunner) Probe(ctx context.Context, source core.Source) (core.Media, error) {
	return r.runner.Probe(ctx, source)
}
func (r *queuedIdentityRunner) Capabilities(ctx context.Context) (core.Capabilities, error) {
	return r.runner.Capabilities(ctx)
}
func (r *queuedIdentityRunner) Cancel(string) error { return nil }
func (r *queuedIdentityRunner) Start(ctx context.Context, plan core.ExecutionPlan, callbacks core.Callbacks) (core.Result, error) {
	if plan.JobID != "blocker" {
		return r.runner.Start(ctx, plan, callbacks)
	}
	close(r.started)
	select {
	case <-r.release:
		return core.Result{}, nil
	case <-ctx.Done():
		return core.Result{}, ctx.Err()
	}
}
