//go:build !windows

package localstore

import (
	"golang.org/x/sys/unix"
	"os"
	"time"
)

func lockFile(file *os.File) (func(), error) {
	deadline := time.Now().Add(3 * time.Second)
	for {
		err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return func() { unix.Flock(int(file.Fd()), unix.LOCK_UN) }, nil
		}
		if err != unix.EWOULDBLOCK || time.Now().After(deadline) {
			return nil, err
		}
		time.Sleep(10 * time.Millisecond)
	}
}
func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := unix.Kill(pid, 0)
	return err == nil || err == unix.EPERM
}
