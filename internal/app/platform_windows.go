//go:build windows

package app

import (
	"kompeg/internal/core"
	"kompeg/internal/runner/windows"
	"log"
	"os/exec"
	"syscall"
)

func newPlatformRunner() core.Runner {
	config, err := readBinaryConfig()
	if err != nil {
		log.Printf("read FFmpeg configuration: %v", err)
	}
	environment := discoverEnvironment(config)
	return newConfiguredRunner(environment.FFmpeg.Path, environment.FFprobe.Path)
}

func newConfiguredRunner(ffmpeg, ffprobe string) core.Runner { return windows.New(ffmpeg, ffprobe) }
func resolvePlatformBinary(binary string) string             { return windows.ResolveBinary(binary) }
func hidePlatformProcess(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}

func previewBinary(r core.Runner) string { return r.(*windows.Runner).FFmpeg }
