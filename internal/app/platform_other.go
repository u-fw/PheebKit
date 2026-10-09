//go:build !windows && !android

package app

import (
	"kompeg/internal/core"
	androidrunner "kompeg/internal/runner/android"
)

func newPlatformRunner() core.Runner { return androidrunner.New() }

func previewBinary(core.Runner) string { return "ffmpeg" }
