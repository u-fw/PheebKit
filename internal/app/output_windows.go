//go:build windows

package app

import (
	"golang.org/x/sys/windows"
	"path/filepath"
)

func openOutput(path string, folder bool) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	operation, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	target := absolute
	parameters := ""
	if folder {
		target = "explorer.exe"
		parameters = "/select,\"" + absolute + "\""
	}
	file, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	args, err := windows.UTF16PtrFromString(parameters)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, operation, file, args, nil, windows.SW_SHOWNORMAL)
}
