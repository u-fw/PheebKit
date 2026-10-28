package jobs

import (
	"errors"
	"fmt"
	"pheebkit/internal/core"
	"sync/atomic"
	"testing"
	"time"
)

func TestBatchPersistsOneCompleteSnapshot(t *testing.T) {
	manager := New(&testRunner{}, nil)
	manager.SetQueuePaused(true)
	defer manager.Close()
	var calls atomic.Int32
	manager.Restore(nil, func(jobs []core.Job) error {
		call := calls.Add(1)
		if call == 1 && len(jobs) != 32 {
			t.Error("persisted a partial batch")
		}
		return nil
	})
	plans := make([]core.ExecutionPlan, 32)
	for i := range plans {
		plans[i].JobID = fmt.Sprintf("batch-%d", i)
	}
	if _, err := manager.EnqueueBatch(plans, nil); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("persisted identical batch %d times", got)
	}
	deadline = time.Now().Add(2 * time.Second)
	for {
		manager.mu.Lock()
		allSaved := true
		for _, id := range manager.order {
			if manager.savedStates[id] != core.JobQueued {
				allSaved = false
				break
			}
		}
		manager.mu.Unlock()
		if allSaved {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("state snapshot was not marked persisted")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestPersistenceDoesNotBlockQueueAdmission(t *testing.T) {
	manager := New(&testRunner{}, nil)
	manager.SetQueuePaused(true)
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	manager.Restore(nil, func([]core.Job) error {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
		return nil
	})
	if _, err := manager.Enqueue(core.ExecutionPlan{JobID: "first"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("persistence did not start")
	}
	admitted := make(chan error, 1)
	go func() {
		_, err := manager.Enqueue(core.ExecutionPlan{JobID: "second"})
		admitted <- err
	}()
	select {
	case err := <-admitted:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(250 * time.Millisecond):
		close(release)
		manager.Close()
		t.Fatal("queue admission waited for disk persistence")
	}
	close(release)
	manager.Close()
}

func TestPersistenceCoalescesToTheLatestCompletedState(t *testing.T) {
	runner := &testRunner{started: make(chan string, 1), release: make(chan struct{})}
	close(runner.release)
	manager := New(runner, nil)
	manager.SetQueuePaused(true)
	entered := make(chan struct{}, 1)
	releaseWrite := make(chan struct{})
	snapshots := make(chan []core.Job, 8)
	manager.Restore(nil, func(jobs []core.Job) error {
		snapshots <- jobs
		select {
		case entered <- struct{}{}:
			<-releaseWrite
		default:
		}
		return nil
	})
	if _, err := manager.Enqueue(core.ExecutionPlan{JobID: "coalesced"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("initial persistence did not start")
	}
	manager.SetQueuePaused(false)
	awaitStart(t, runner.started, "coalesced")
	awaitState(t, manager, "coalesced", core.JobCompleted)
	close(releaseWrite)
	deadline := time.After(2 * time.Second)
	for {
		select {
		case jobs := <-snapshots:
			if len(jobs) == 1 && jobs[0].State == core.JobCompleted {
				manager.Close()
				return
			}
		case <-deadline:
			manager.Close()
			t.Fatal("latest completed state was not persisted after the slow write")
		}
	}
}

func TestPersistenceRetriesWithBackoffAndFlushesOnClose(t *testing.T) {
	manager := New(&testRunner{}, nil)
	manager.SetQueuePaused(true)
	var calls atomic.Int32
	manager.Restore(nil, func([]core.Job) error {
		calls.Add(1)
		return errors.New("disk unavailable")
	})
	if _, err := manager.Enqueue(core.ExecutionPlan{JobID: "retry"}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for manager.PersistenceError() == "" && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if manager.PersistenceError() == "" {
		t.Fatal("persistence failure was not surfaced")
	}
	failedCalls := calls.Load()
	time.Sleep(100 * time.Millisecond)
	if calls.Load() != failedCalls {
		t.Fatal("failed persistence retried without backoff")
	}
	manager.Close()
	if calls.Load() <= failedCalls {
		t.Fatal("close did not attempt a final history flush")
	}
}
