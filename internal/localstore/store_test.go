package localstore

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
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
