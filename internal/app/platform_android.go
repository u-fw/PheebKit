//go:build android

package app

import (
	"kompeg/internal/core"
	androidrunner "kompeg/internal/runner/android"
	"os/exec"
)

func newPlatformRunner() core.Runner { return androidrunner.New() }

func previewBinary(core.Runner) string            { return "ffmpeg" }
func newConfiguredRunner(_, _ string) core.Runner { return androidrunner.New() }
func resolvePlatformBinary(binary string) string  { return binary }
func hidePlatformProcess(_ *exec.Cmd)             {}
