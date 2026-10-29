package localstore

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestConcurrentAtomicUpdatesAndCorruption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := struct{ Count int }{}
			if err := Update(path, &data, func() error { data.Count++; return nil }); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	data := struct{ Count int }{}
	if err := Read(path, &data); err != nil || data.Count != 16 {
		t.Fatalf("lost concurrent write: %+v %v", data, err)
	}
	corrupt := []byte("{broken")
	if err := os.WriteFile(path, corrupt, 0600); err != nil {
		t.Fatal(err)
	}
	if err := Update(path, &data, func() error { data.Count++; return nil }); err == nil {
		t.Fatal("corruption silently overwritten")
	}
	contents, _ := os.ReadFile(path)
	if string(contents) != string(corrupt) {
		t.Fatal("corrupt original changed")
	}
}

func TestSlowLocalWriterDoesNotConsumeCrossProcessWaitBudget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "slow.json")
	entered, release := make(chan struct{}), make(chan struct{})
	var unblock sync.Once
	defer unblock.Do(func() { close(release) })
	finished := make(chan error, 2)
	go func() {
		data := struct{ Count int }{}
		finished <- Update(path, &data, func() error { close(entered); <-release; data.Count++; return nil })
	}()
	<-entered
	go func() {
		data := struct{ Count int }{}
		finished <- Update(path, &data, func() error { data.Count++; return nil })
	}()
	// Hold the first writer longer than the 3-second external lock deadline.
	select {
	case err := <-finished:
		t.Fatalf("local waiter escaped or consumed the external lock budget: %v", err)
	case <-time.After(3500 * time.Millisecond):
	}
	unblock.Do(func() { close(release) })
	for i := 0; i < 2; i++ {
		select {
		case err := <-finished:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("queued local writer did not finish")
		}
	}
	data := struct{ Count int }{}
	if err := Read(path, &data); err != nil || data.Count != 2 {
		t.Fatalf("queued write was lost: %+v %v", data, err)
	}
	processLocks.Lock()
	defer processLocks.Unlock()
	if len(processLocks.paths) != 0 {
		t.Fatal("idle path locks leaked")
	}
}

func TestIndependentFilesDoNotShareTheLocalWriteQueue(t *testing.T) {
	root := t.TempDir()
	release, err := lockPath(filepath.Join(root, "one.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	done := make(chan error, 1)
	go func() {
		value := struct{ Count int }{}
		done <- Update(filepath.Join(root, "two.json"), &value, func() error { value.Count++; return nil })
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("an unrelated file waited for the global path registry lock")
	}
}
