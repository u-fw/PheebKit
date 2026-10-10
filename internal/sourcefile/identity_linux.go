//go:build linux || android

package sourcefile

import (
	"fmt"
	"os"
	"syscall"
)

func platformIdentity(_ *os.File, info os.FileInfo) (string, error) {
	stat := info.Sys().(*syscall.Stat_t)
	return fmt.Sprintf("%d:%d:%d:%d", stat.Dev, stat.Ino, stat.Ctim.Sec, stat.Ctim.Nsec), nil
}
