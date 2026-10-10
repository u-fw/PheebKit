//go:build windows

package main

import "golang.org/x/sys/windows"

// Presence only: multiple application windows remain allowed. Installers use
// this mutex to avoid replacing binaries while media work could be running.
func holdInstallerGuard() (func(), error) {
	name, err := windows.UTF16PtrFromString(`Local\kompeg-application-running`)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateMutex(nil, false, name)
	if err != nil && err != windows.ERROR_ALREADY_EXISTS {
		return nil, err
	}
	return func() { windows.CloseHandle(handle) }, nil
}
