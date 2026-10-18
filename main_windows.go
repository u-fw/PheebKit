//go:build windows

package main

import (
	"golang.org/x/sys/windows"
	"pheebkit/internal/compat"
)

// Presence only: multiple application windows remain allowed. Installers use
// this mutex to avoid replacing binaries while media work could be running.
func holdInstallerGuard() (func(), error) {
	var handles []windows.Handle
	closeHandles := func() {
		for _, handle := range handles {
			windows.CloseHandle(handle)
		}
	}
	for _, value := range []string{`Local\PheebKit-application-running`, compat.LegacyRunningMutex} {
		name, err := windows.UTF16PtrFromString(value)
		if err != nil {
			closeHandles()
			return nil, err
		}
		handle, err := windows.CreateMutex(nil, false, name)
		if err != nil && err != windows.ERROR_ALREADY_EXISTS {
			closeHandles()
			return nil, err
		}
		handles = append(handles, handle)
	}
	return closeHandles, nil
}
