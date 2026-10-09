//go:build windows

package windows

import (
	"path/filepath"
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
	// Extended Win32 paths disable normalization, so forward slashes and dot
	// components must be normalized before applying the long-path prefix.
	path = filepath.Clean(path)
	if strings.HasPrefix(path, `\\?\`) {
		return path
	}
	if strings.HasPrefix(path, `\\`) {
		return `\\?\UNC\` + strings.TrimPrefix(path, `\\`)
	}
	return `\\?\` + path
}
