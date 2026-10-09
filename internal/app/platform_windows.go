//go:build windows

package app

import (
	"kompeg/internal/core"
	"kompeg/internal/runner/windows"
	"os"
)

func newPlatformRunner() core.Runner {
	return windows.New(os.Getenv("KOMPEG_FFMPEG"), os.Getenv("KOMPEG_FFPROBE"))
}

func previewBinary(r core.Runner) string { return r.(*windows.Runner).FFmpeg }
