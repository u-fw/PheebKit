package app

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"pheebkit/internal/core"
	"pheebkit/internal/jobs"
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
	if err := service.ConfirmQuit(false); err != nil {
		t.Fatal(err)
	}
	current, _ := service.jobs.Get(job.ID)
	if current.State != core.JobCancelled {
		t.Fatalf("quit did not cancel: %+v", current)
	}
	if !service.CanClose() {
		t.Fatal("confirmed quit blocked")
	}
}

type historyCloseHost struct {
	NativeHost
	requests chan CloseRequest
	quits    atomic.Int32
}

func (h *historyCloseHost) Emit(name string, values ...interface{}) {
	if name == "app:close-requested" {
		h.requests <- values[0].(CloseRequest)
	}
}
func (h *historyCloseHost) Quit()                              { h.quits.Add(1) }
func (h *historyCloseHost) SetTaskbarProgress(TaskbarProgress) {}

func TestIdleCloseReportsFailedHistoryAndAllowsExplicitRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	host := &historyCloseHost{requests: make(chan CloseRequest, 16)}
	service := &Service{ctx: ctx, cancel: cancel, jobs: jobs.New(&lifecycleRunner{}, nil), app: host}
	defer service.Close()
	var fail atomic.Bool
	fail.Store(true)
	service.jobs.Restore(nil, func([]core.Job) error {
		if fail.Load() {
			return errors.New("disk full")
		}
		return nil
	})
	if service.CanClose() {
		t.Fatal("final history failure allowed silent native exit")
	}
	select {
	case request := <-host.requests:
		if !request.HistorySaveFailed || !request.ShutdownStarted {
			t.Fatalf("history failure was not surfaced: %+v", request)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("history warning did not arrive")
	}
	if err := service.ConfirmQuit(false); err == nil || host.quits.Load() != 0 {
		t.Fatal("failed retry quit without acknowledgement")
	}
	if service.CanClose() {
		t.Fatal("a second native close bypassed the history warning")
	}
	fail.Store(false)
	if err := service.ConfirmQuit(false); err != nil {
		t.Fatal(err)
	}
	if host.quits.Load() != 1 || !service.CanClose() {
		t.Fatal("successful history retry did not allow exit")
	}
}

func TestHistoryLossRequiresExplicitAcknowledgement(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	host := &historyCloseHost{requests: make(chan CloseRequest, 16)}
	service := &Service{ctx: ctx, cancel: cancel, jobs: jobs.New(&lifecycleRunner{}, nil), app: host}
	defer service.Close()
	service.jobs.Restore(nil, func([]core.Job) error { return errors.New("read-only disk") })
	if err := service.ConfirmQuit(false); err == nil {
		t.Fatal("failed history write was ignored")
	}
	if err := service.ConfirmQuit(true); err != nil || host.quits.Load() != 1 || !service.CanClose() {
		t.Fatalf("explicit history-loss acknowledgement did not exit: %v", err)
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
	service.CanClose()
	awaitServiceShutdown(t, service)
	if !service.CanClose() {
		t.Fatal("idle close blocked after history flush")
	}
}

func awaitServiceShutdown(t *testing.T, service *Service) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		service.lifecycleMu.Lock()
		finished := service.shutdownFinished
		service.lifecycleMu.Unlock()
		if finished {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("service shutdown did not finish")
}

func TestIdleCloseDoesNotBlockNativeWindowOnHistoryIO(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	host := &historyCloseHost{requests: make(chan CloseRequest, 16)}
	service := &Service{ctx: ctx, cancel: cancel, jobs: jobs.New(&lifecycleRunner{}, nil), app: host}
	defer service.Close()
	entered, release := make(chan struct{}), make(chan struct{})
	service.jobs.Restore(nil, func([]core.Job) error { close(entered); <-release; return nil })
	defer close(release)
	result := make(chan bool, 1)
	go func() { result <- service.CanClose() }()
	select {
	case allowed := <-result:
		if allowed {
			t.Fatal("native close committed before history flush")
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("native close hook waited on history I/O")
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("asynchronous history flush did not start")
	}
	if service.CanClose() || host.quits.Load() != 0 {
		t.Fatal("second native close bypassed an in-flight flush")
	}
}
