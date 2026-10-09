//go:build windows

package windows

import (
	"strings"
	"syscall"
)

func publishFile(from, to string) error {
	source, err := syscall.UTF16PtrFromString(extendedPath(from))
	if err != nil {
		return err
	}
	destination, err := syscall.UTF16PtrFromString(extendedPath(to))
	if err != nil {
		return err
	}
	// MoveFile fails if the destination already exists; unlike os.Rename it does
	// not replace another process's file on Windows.
	return syscall.MoveFile(source, destination)
}

func extendedPath(path string) string {
	if strings.HasPrefix(path, `\\?\`) {
		return path
	}
	if strings.HasPrefix(path, `\\`) {
		return `\\?\UNC\` + strings.TrimPrefix(path, `\\`)
	}
	return `\\?\` + path
}
