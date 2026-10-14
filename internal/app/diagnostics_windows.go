//go:build windows

package app

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func diagnosticOSVersion() string {
	version := windows.RtlGetVersion()
	return fmt.Sprintf("%d.%d.%d", version.MajorVersion, version.MinorVersion, version.BuildNumber)
}
