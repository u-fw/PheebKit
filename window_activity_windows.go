//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	activityUser32          = windows.NewLazySystemDLL("user32.dll")
	procGetForegroundWindow = activityUser32.NewProc("GetForegroundWindow")
	procGetWindowThreadPID  = activityUser32.NewProc("GetWindowThreadProcessId")
)

func applicationIsForeground() bool {
	window, _, _ := procGetForegroundWindow.Call()
	if window == 0 {
		return false
	}
	var processID uint32
	procGetWindowThreadPID.Call(
		window,
		uintptr(unsafe.Pointer(&processID)),
	)
	return processID == windows.GetCurrentProcessId()
}
