package jobs

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"kompeg/internal/core"
)

func TestPauseReorderAndCancelPreserveRunningTask(t *testing.T) {
	runner := &testRunner{started: make(chan string, 4), release: make(chan struct{})}
	m := New(runner, nil)
	defer m.Close()
	if _, err := m.Enqueue(core.ExecutionPlan{JobID: "running"}); err != nil {
		t.Fatal(err)
	}
	awaitStart(t, runner.started, "running")
	for _, id := range []string{"first", "second", "third"} {
		if _, err := m.Enqueue(core.ExecutionPlan{JobID: id}); err != nil {
			t.Fatal(err)
		}
	}
	initial := m.QueueStatus()
	paused, err := m.SetQueuePaused(true)
	if err != nil || !paused.Paused || paused.Revision <= initial.Revision {
		t.Fatalf("pause: %+v %v", paused, err)
	}
	if current, _ := m.Get("running"); current.State != core.JobRunning {
		t.Fatal("pause altered the currently running task")
	}
	moved, err := m.MoveQueued("third", -1)
	if err != nil || !reflect.DeepEqual(moved.PendingIDs, []string{"first", "third", "second"}) {
		t.Fatalf("move: %+v %v", moved, err)
	}
	if err := m.Cancel("first"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.MoveQueued("running", 1); err == nil {
		t.Fatal("reordered a task that already started")
	}
	if _, err := m.MoveQueued("first", 1); err == nil {
		t.Fatal("reordered a cancelled task")
	}
	if _, err := m.MoveQueued("third", 0); err == nil {
		t.Fatal("accepted invalid direction")
	}
	if _, err := m.MoveQueued("third", -1); err == nil {
		t.Fatal("accepted move past the queue boundary")
	}
	// Finishing the running job while paused must not start the next one.
	close(runner.release)
	awaitState(t, m, "running", core.JobCompleted)
	select {
	case id := <-runner.started:
		t.Fatalf("paused queue started %s", id)
	case <-time.After(40 * time.Millisecond):
	}
	status := m.QueueStatus()
	status.PendingIDs[0] = "external mutation"
	if m.QueueStatus().PendingIDs[0] != "third" {
		t.Fatal("status exposed the scheduler slice")
	}
	if _, err := m.SetQueuePaused(false); err != nil {
		t.Fatal(err)
	}
	awaitStart(t, runner.started, "third")
	awaitStart(t, runner.started, "second")
	awaitState(t, m, "second", core.JobCompleted)
	if runner.max.Load() != 1 || runner.starts.Load() != 3 {
		t.Fatal("scheduler overlapped jobs or executed a cancelled job")
	}
}

func TestBatchAdmissionIsAtomicForCapacityIDsAndOutputs(t *testing.T) {
	runner := &testRunner{started: make(chan string, 1), release: make(chan struct{})}
	m := New(runner, nil)
	defer m.Close()
	m.SetQueuePaused(true)
	dir := t.TempDir()
	plan := func(id, file string) core.ExecutionPlan {
		return core.ExecutionPlan{JobID: id, Output: core.Source{URI: filepath.Join(dir, file)}}
	}
	if _, err := m.EnqueueBatch([]core.ExecutionPlan{plan("a", "out.mp4"), plan("b", "out.mp4")}, nil); err == nil {
		t.Fatal("duplicate output admitted")
	}
	if len(m.List()) != 0 {
		t.Fatal("duplicate output left a partial batch")
	}
	if _, err := m.EnqueueBatch([]core.ExecutionPlan{plan("a", "one.mp4"), plan("a", "two.mp4")}, nil); err == nil {
		t.Fatal("duplicate ID admitted")
	}
	if len(m.List()) != 0 {
		t.Fatal("duplicate ID left a partial batch")
	}
	if err := os.WriteFile(filepath.Join(dir, "exists.mp4"), []byte("owned by user"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := m.EnqueueBatch([]core.ExecutionPlan{plan("one", "new.mp4"), plan("two", "exists.mp4")}, nil); err == nil {
		t.Fatal("existing output admitted")
	}
	if len(m.List()) != 0 {
		t.Fatal("existing output left a partial batch")
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "exists.mp4")); string(data) != "owned by user" {
		t.Fatal("admission touched an existing output")
	}
	if _, err := m.Enqueue(plan("reserved", "reserved.mp4")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.EnqueueBatch([]core.ExecutionPlan{plan("other", "other.mp4"), plan("duplicate", "reserved.mp4")}, nil); err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatal("active output reservation ignored", err)
	}
	if len(m.List()) != 1 {
		t.Fatal("reservation failure left a partial batch")
	}
	for i := 0; i < 30; i++ {
		if _, err := m.Enqueue(plan(fmt.Sprintf("fill-%d", i), fmt.Sprintf("fill-%d.mp4", i))); err != nil {
			t.Fatal(err)
		}
	}
	before := m.QueueStatus()
	if _, err := m.EnqueueBatch([]core.ExecutionPlan{plan("capacity-a", "ca.mp4"), plan("capacity-b", "cb.mp4")}, nil); err == nil {
		t.Fatal("partially fitting batch admitted")
	}
	if after := m.QueueStatus(); !reflect.DeepEqual(before, after) || len(m.List()) != 31 {
		t.Fatal("rejected batch mutated scheduler state")
	}
	if err := m.Cancel("reserved"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Enqueue(plan("replacement", "reserved.mp4")); err != nil {
		t.Fatal("cancelled waiting job kept its reservation", err)
	}
	if runner.starts.Load() != 0 {
		t.Fatal("paused admission started a task")
	}
}

func TestBatchPrunesHistoryAndClonesNestedIntent(t *testing.T) {
	runner := &testRunner{started: make(chan string, 1), release: make(chan struct{})}
	m := New(runner, nil)
	defer m.Close()
	m.SetQueuePaused(true)
	history := make([]core.Job, 195)
	for i := range history {
		history[i] = core.Job{ID: fmt.Sprintf("old-%d", i), State: core.JobCompleted}
	}
	m.Restore(history, nil)
	plans := make([]core.ExecutionPlan, 20)
	projects := make([]core.Project, 20)
	for i := range plans {
		plans[i].JobID = fmt.Sprintf("new-%d", i)
		projects[i] = core.Project{Settings: core.DefaultSettings()}
		projects[i].Settings.AudioSelection = &core.TrackSelection{Mode: "selected", Indices: []int{3}}
		projects[i].Settings.Transform = &core.VideoTransform{Crop: &core.CropMargins{Top: 2}}
	}
	batch, err := m.EnqueueBatch(plans, projects)
	if err != nil || len(m.List()) != historyLimit || len(batch) != 20 {
		t.Fatalf("history pruning: %d %+v %v", len(m.List()), batch, err)
	}
	projects[0].Settings.AudioSelection.Indices[0] = 99
	projects[0].Settings.Transform.Crop.Top = 99
	batch[0].Project.Settings.AudioSelection.Indices[0] = 88
	batch[0].Project.Settings.Transform.Crop.Top = 88
	again, _ := m.Get("new-0")
	if again.Project.Settings.AudioSelection.Indices[0] != 3 || again.Project.Settings.Transform.Crop.Top != 2 {
		t.Fatal("nested intent shared with callers")
	}
	m.Close()
	if len(m.QueueStatus().PendingIDs) != 0 {
		t.Fatal("closed scheduler retained pending work")
	}
	for _, job := range m.List() {
		if job.State == core.JobQueued {
			t.Fatal("closing paused scheduler did not cancel waiting work")
		}
	}
	if _, err := m.SetQueuePaused(false); err == nil {
		t.Fatal("closed scheduler accepted resume")
	}
}

func TestCancelAfterSchedulerClaimBeforeRunnerStart(t *testing.T) {
	runner := &testRunner{started: make(chan string, 1), release: make(chan struct{})}
	m := New(runner, nil)
	defer m.Close()
	m.SetQueuePaused(true)
	if _, err := m.Enqueue(core.ExecutionPlan{JobID: "claimed"}); err != nil {
		t.Fatal(err)
	}
	// Simulate the scheduler's atomic claim, then deliver cancellation at the
	// exact boundary before run can attach a context. No timing guess is needed.
	m.mu.Lock()
	m.pending = nil
	m.records["claimed"].job.State = core.JobPreparing
	m.mu.Unlock()
	if err := m.Cancel("claimed"); err != nil {
		t.Fatal(err)
	}
	m.run("claimed")
	current, _ := m.Get("claimed")
	if current.State != core.JobCancelled || current.CompletedAt.IsZero() || runner.starts.Load() != 0 {
		t.Fatalf("claimed cancellation stuck or started runner: %+v", current)
	}
}
