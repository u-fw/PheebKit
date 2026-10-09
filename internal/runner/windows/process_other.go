//go:build !windows

package windows

import "os/exec"

// Allows core tests to build on other hosts; this adapter targets Windows.
func hideProcess(_ *exec.Cmd) {}
