package app

import (
	"context"
	"testing"

	"kompeg/internal/core"
	"kompeg/internal/jobs"
)

type lifecycleRunner struct{ started chan struct{} }

func (r *lifecycleRunner) Probe(context.Context, core.Source) (core.Media, error) {
	return core.Media{}, nil
}
func (r *lifecycleRunner) Capabilities(context.Context) (core.Capabilities, error) {
	return core.Capabilities{}, nil
}
func (r *lifecycleRunner) Cancel(string) error { return nil }
func (r *lifecycleRunner) Start(ctx context.Context, _ core.ExecutionPlan, _ core.Callbacks) (core.Result, error) {
	close(r.started)
	<-ctx.Done()
	return core.Result{}, ctx.Err()
}

func TestCloseGuardAndExplicitCancellation(t *testing.T) {
	runner := &lifecycleRunner{started: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	service := &Service{ctx: ctx, cancel: cancel, jobs: jobs.New(runner, nil)}
	defer service.Close()
	project := core.Project{Settings: core.DefaultSettings()}
	job, err := service.jobs.EnqueueProject(core.ExecutionPlan{}, project)
	if err != nil {
		t.Fatal(err)
	}
	<-runner.started
	if service.CanClose() {
		t.Fatal("active job allowed accidental close")
	}
	if _, err := service.ReuseJob(job.ID); err == nil {
		t.Fatal("active job reused")
	}
	service.ConfirmQuit()
	current, _ := service.jobs.Get(job.ID)
	if current.State != core.JobCancelled {
		t.Fatalf("quit did not cancel: %+v", current)
	}
	if !service.CanClose() {
		t.Fatal("confirmed quit blocked")
	}
}

func TestDownloadAndIdleCloseGuard(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := &lifecycleRunner{started: make(chan struct{})}
	service := &Service{ctx: ctx, cancel: cancel, jobs: jobs.New(runner, nil), setupStatus: FFmpegSetupStatus{Active: true}}
	service.setupBusy.Store(true)
	defer service.Close()
	if service.CanClose() {
		t.Fatal("download allowed accidental close")
	}
	service.setupStatus.Active = false
	service.setupBusy.Store(false)
	if !service.CanClose() || !service.exiting {
		t.Fatal("idle close blocked")
	}
}
