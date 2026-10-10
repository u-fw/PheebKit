//go:build !windows

package main

func holdInstallerGuard() (func(), error) { return func() {}, nil }
