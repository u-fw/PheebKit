//go:build !windows

package bundledffmpeg

import (
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

func acquireRuntimeLock(root, name string) (func(), error) {
	file, err := os.OpenFile(filepath.Join(root, name), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(5 * time.Minute)
	for {
		err = unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return func() {
				_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
				_ = file.Close()
			}, nil
		}
		if err != unix.EWOULDBLOCK || time.Now().After(deadline) {
			_ = file.Close()
			return nil, err
		}
		time.Sleep(20 * time.Millisecond)
	}
}
