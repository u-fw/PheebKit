package localstore

import (
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type pathLock struct {
	mu    sync.Mutex
	users int
}

var processLocks = struct {
	sync.Mutex
	paths map[string]*pathLock
}{paths: make(map[string]*pathLock)}

// Same-process writers queue before starting the bounded cross-process lock
// wait. References include waiters, and idle entries are removed after use.
func lockPath(path string) (func(), error) {
	key, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if parent, err := filepath.EvalSymlinks(filepath.Dir(key)); err == nil {
		key = filepath.Join(parent, filepath.Base(key))
	}
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	processLocks.Lock()
	entry := processLocks.paths[key]
	if entry == nil {
		entry = &pathLock{}
		processLocks.paths[key] = entry
	}
	entry.users++
	processLocks.Unlock()
	entry.mu.Lock()
	return func() {
		entry.mu.Unlock()
		processLocks.Lock()
		entry.users--
		if entry.users == 0 {
			delete(processLocks.paths, key)
		}
		processLocks.Unlock()
	}, nil
}
