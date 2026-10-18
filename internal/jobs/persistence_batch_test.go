package jobs

import (
	"fmt"
	"pheebkit/internal/core"
	"testing"
)

func TestBatchPersistsOneCompleteSnapshot(t *testing.T) {
	manager := New(&testRunner{}, nil)
	manager.SetQueuePaused(true)
	defer manager.Close()
	calls := 0
	manager.Restore(nil, func(jobs []core.Job) error {
		calls++
		if calls == 1 && len(jobs) != 32 {
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
	if calls != 1 {
		t.Fatalf("persisted identical batch %d times", calls)
	}
	manager.mu.Lock()
	for _, id := range manager.order {
		if manager.savedStates[id] != core.JobQueued {
			t.Error("state snapshot was not marked persisted")
		}
	}
	manager.mu.Unlock()
}
