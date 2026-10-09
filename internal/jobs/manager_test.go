package jobs

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"ffmpegstudio/internal/core"
)

type testRunner struct {
	started chan string
	release chan struct{}
	fail    bool
	starts  atomic.Int32
	active  atomic.Int32
	max     atomic.Int32
}

func (*testRunner) Probe(context.Context, core.Source) (core.Media, error) {
	return core.Media{}, core.ErrUnsupported
}
func (*testRunner) Capabilities(context.Context) (core.Capabilities, error) {
	return core.Capabilities{}, nil
}
func (*testRunner) Cancel(string) error { return nil }
func (r *testRunner) Start(ctx context.Context, plan core.ExecutionPlan, cb core.Callbacks) (core.Result, error) {
	r.starts.Add(1)
	active := r.active.Add(1)
	if active > r.max.Load() {
		r.max.Store(active)
	}
	defer r.active.Add(-1)
	r.started <- plan.JobID
	for i := 0; i < 300; i++ {
		cb.Log(strings.Repeat("log ", 700))
		cb.Progress(core.Progress{Percent: 100, Finished: true, TotalTimeSeconds: 5})
	}
	if r.fail {
		return core.Result{}, errors.New("encoder rejected input")
	}
	select {
	case <-ctx.Done():
		return core.Result{}, ctx.Err()
	case <-r.release:
		return core.Result{InputSizeBytes: 1000, OutputSizeBytes: 500, SavedPercent: 50}, nil
	}
}

func TestQueueCancellationAndBoundedImmutableSnapshots(t *testing.T) {
	runner := &testRunner{started: make(chan string, 3), release: make(chan struct{})}
	manager := New(runner, nil)
	defer manager.Close()
	first, err := manager.Enqueue(core.ExecutionPlan{JobID: "first", DurationSeconds: 5})
	if err != nil || first.State != core.JobQueued {
		t.Fatalf("enqueue: %+v %v", first, err)
	}
	awaitStart(t, runner.started, "first")
	if _, err := manager.Enqueue(core.ExecutionPlan{JobID: "second"}); err != nil {
		t.Fatal(err)
	}
	if err := manager.Cancel("second"); err != nil {
		t.Fatal(err)
	}
	if err := manager.Cancel("first"); err != nil {
		t.Fatal(err)
	}
	job := awaitState(t, manager, "first", core.JobCancelled)
	if runner.starts.Load() != 1 || runner.max.Load() != 1 {
		t.Fatalf("cancelled queued job executed or jobs overlapped: starts=%d max=%d", runner.starts.Load(), runner.max.Load())
	}
	if job.Progress.Percent >= 100 {
		t.Fatal("encoder progress marked a cancelled job complete")
	}
	totalBytes := 0
	for _, line := range job.Logs {
		totalBytes += len(line)
	}
	if totalBytes > logByteLimit || len(job.Logs) > logLineLimit {
		t.Fatalf("logs are unbounded: %d bytes, %d lines", totalBytes, len(job.Logs))
	}
	if len(job.Logs) > 0 {
		job.Logs[0] = "mutated frontend snapshot"
		again, _ := manager.Get("first")
		if again.Logs[0] == job.Logs[0] {
			t.Fatal("frontend can mutate manager log storage")
		}
	}
	awaitState(t, manager, "second", core.JobCancelled)
}

func TestCompletionFailureAndClose(t *testing.T) {
	for _, fail := range []bool{false, true} {
		runner := &testRunner{started: make(chan string, 1), release: make(chan struct{}), fail: fail}
		close(runner.release)
		manager := New(runner, nil)
		job, err := manager.Enqueue(core.ExecutionPlan{})
		if err != nil {
			t.Fatal(err)
		}
		state := core.JobCompleted
		if fail {
			state = core.JobFailed
		}
		job = awaitState(t, manager, job.ID, state)
		if fail && !strings.Contains(job.Error, "encoder rejected") {
			t.Fatalf("lost encoder failure: %+v", job)
		}
		if !fail && (job.Result == nil || job.Result.SavedPercent != 50 || job.Progress.Percent != 100) {
			t.Fatalf("lost completion result: %+v", job)
		}
		manager.Close()
		manager.Close()
		if _, err := manager.Enqueue(core.ExecutionPlan{}); err == nil {
			t.Fatal("closed manager accepted a job")
		}
	}
}

func awaitStart(t *testing.T, started <-chan string, id string) {
	t.Helper()
	select {
	case got := <-started:
		if got != id {
			t.Fatalf("started %s, want %s", got, id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runner did not start")
	}
}

func awaitState(t *testing.T, manager *Manager, id, state string) core.Job {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		job, exists := manager.Get(id)
		if exists && job.State == state {
			return job
		}
		time.Sleep(5 * time.Millisecond)
	}
	job, _ := manager.Get(id)
	t.Fatalf("job %s reached %s, want %s", id, job.State, state)
	return core.Job{}
}
