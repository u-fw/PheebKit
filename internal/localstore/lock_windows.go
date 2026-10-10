//go:build windows

package localstore

import (
	"golang.org/x/sys/windows"
	"os"
	"time"
)

func lockFile(file *os.File) (func(), error) {
	overlap := new(windows.Overlapped)
	deadline := time.Now().Add(3 * time.Second)
	for {
		err := windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, overlap)
		if err == nil {
			return func() { windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, overlap) }, nil
		}
		if err != windows.ERROR_LOCK_VIOLATION || time.Now().After(deadline) {
			return nil, err
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return err == windows.ERROR_ACCESS_DENIED
	}
	defer windows.CloseHandle(handle)
	var code uint32
	return windows.GetExitCodeProcess(handle, &code) == nil && code == 259
}
