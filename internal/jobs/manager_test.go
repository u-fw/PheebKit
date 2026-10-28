package jobs

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"pheebkit/internal/core"
)

type testRunner struct {
	started chan string
	release chan struct{}
	fail    bool
	starts  atomic.Int32
	active  atomic.Int32
	max     atomic.Int32
}

func TestExpectedSourceChannelsSnapshotIsIsolated(t *testing.T) {
	m := New(&testRunner{}, nil)
	defer m.Close()
	m.SetQueuePaused(true)
	p := core.ExecutionPlan{ExpectedOutput: &core.OutputExpectation{AudioSourceChannels: []int{6}}}
	job, err := m.Enqueue(p)
	if err != nil {
		t.Fatal(err)
	}
	p.ExpectedOutput.AudioSourceChannels[0] = 2
	m.mu.Lock()
	channels := m.records[job.ID].plan.ExpectedOutput.AudioSourceChannels[0]
	m.mu.Unlock()
	if channels != 6 {
		t.Fatal("caller changed queued source channel expectation")
	}
}

func TestVerificationHistorySnapshotsAreIsolated(t *testing.T) {
	manager := New(&testRunner{}, nil)
	defer manager.Close()
	original := core.Job{ID: "verified", State: core.JobCompleted, Result: &core.Result{Verification: &core.OutputVerification{VideoStreams: 1, Warnings: []string{"check duration"}}}}
	manager.Restore([]core.Job{original}, nil)
	original.Result.Verification.Warnings[0] = "caller mutation"
	got, ok := manager.Get("verified")
	if !ok || got.Result.Verification.Warnings[0] != "check duration" {
		t.Fatal("restore retained caller-owned warnings")
	}
	got.Result.Verification.Warnings[0] = "snapshot mutation"
	again, _ := manager.Get("verified")
	if again.Result.Verification.Warnings[0] != "check duration" {
		t.Fatal("returned snapshot mutated history")
	}
}

func TestJobEventsCarryMonotonicSequenceForRecovery(t *testing.T) {
	events := make(chan core.Job, 4)
	manager := New(&testRunner{}, func(job core.Job) { events <- job })
	defer manager.Close()
	manager.SetQueuePaused(true)
	job, err := manager.Enqueue(core.ExecutionPlan{JobID: "sequenced"})
	if err != nil {
		t.Fatal(err)
	}
	first := <-events
	if err := manager.Cancel(job.ID); err != nil {
		t.Fatal(err)
	}
	second := <-events
	if first.EventSequence == 0 || second.EventSequence != first.EventSequence+1 {
		t.Fatalf("non-contiguous event sequence: %d then %d", first.EventSequence, second.EventSequence)
	}
	stored, ok := manager.Get(job.ID)
	if !ok || stored.EventSequence != second.EventSequence {
		t.Fatalf("snapshot did not expose the latest event sequence: %+v", stored)
	}
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

func TestTrimQueueSnapshotsDoNotShareMutableIntent(t *testing.T) {
	runner := &testRunner{started: make(chan string, 1), release: make(chan struct{})}
	manager := New(runner, nil)
	defer manager.Close()
	trim := &core.TimeRange{StartSeconds: 2.5, EndSeconds: 3.75}
	job, err := manager.Enqueue(core.ExecutionPlan{JobID: "trim", DurationSeconds: 1.25, Trim: trim})
	if err != nil {
		t.Fatal(err)
	}
	trim.EndSeconds = 99
	job.Trim.StartSeconds = 99
	again, _ := manager.Get(job.ID)
	if again.Trim.StartSeconds != 2.5 || again.Trim.EndSeconds != 3.75 {
		t.Fatalf("external intent mutation changed queued range: %+v", again.Trim)
	}
	again.Trim.EndSeconds = 100
	latest, _ := manager.Get(job.ID)
	if latest.Trim.EndSeconds != 3.75 {
		t.Fatal("Get exposes mutable queue trim storage")
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
