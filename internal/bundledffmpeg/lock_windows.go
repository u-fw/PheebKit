//go:build windows

package bundledffmpeg

import (
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
)

func acquireRuntimeLock(root, name string) (func(), error) {
	file, err := os.OpenFile(filepath.Join(root, name), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	overlap := new(windows.Overlapped)
	deadline := time.Now().Add(5 * time.Minute)
	for {
		err = windows.LockFileEx(windows.Handle(file.Fd()),
			windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
			0, 1, 0, overlap)
		if err == nil {
			return func() {
				_ = windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, overlap)
				_ = file.Close()
			}, nil
		}
		if err != windows.ERROR_LOCK_VIOLATION || time.Now().After(deadline) {
			_ = file.Close()
			return nil, err
		}
		time.Sleep(20 * time.Millisecond)
	}
}
