//go:build !windows

package app

func diagnosticOSVersion() string { return "unavailable" }
