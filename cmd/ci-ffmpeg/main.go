// ci-ffmpeg prepares publisher-pinned Windows test tools, never release payloads.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"kompeg/internal/ffmpegsetup"
)

type manifest struct {
	Package       string `json:"package"`
	Version       string `json:"version"`
	ArchiveSHA256 string `json:"archiveSHA256"`
	FFmpeg        string `json:"ffmpeg"`
	FFprobe       string `json:"ffprobe"`
	FFmpegSHA256  string `json:"ffmpegSHA256"`
	FFprobeSHA256 string `json:"ffprobeSHA256"`
}

func digest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
func run() error {
	rootFlag := flag.String("directory", ".tools/ci-ffmpeg", "fresh test-tool directory inside workspace .tools")
	flag.Parse()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return fmt.Errorf("publisher-pinned CI tools currently require Windows x64")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := filepath.Abs(*rootFlag)
	if err != nil {
		return err
	}
	allowed := filepath.Join(cwd, ".tools") + string(filepath.Separator)
	if !strings.HasPrefix(strings.ToLower(root), strings.ToLower(allowed)) {
		return fmt.Errorf("test tools must stay inside this workspace .tools directory")
	}
	manifestPath := filepath.Join(root, "manifest.json")
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		return fmt.Errorf("test manifest already exists or is unreadable; choose a fresh tools directory")
	}
	packages, err := ffmpegsetup.Catalog("stable")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	lastStage := ""
	installer := ffmpegsetup.New(func(progress ffmpegsetup.Progress) {
		if progress.Stage != lastStage {
			fmt.Printf("FFmpeg test tools: %s\n", progress.Stage)
			lastStage = progress.Stage
		}
	})
	result, err := installer.Install(ctx, root, packages)
	if err != nil {
		return err
	}
	ffmpegHash, err := digest(result.FFmpeg)
	if err != nil {
		return err
	}
	ffprobeHash, err := digest(result.FFprobe)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest{Package: packages[0].ID, Version: result.Version, ArchiveSHA256: packages[0].Artifacts[0].SHA256, FFmpeg: result.FFmpeg, FFprobe: result.FFprobe, FFmpegSHA256: ffmpegHash, FFprobeSHA256: ffprobeHash}, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.OpenFile(manifestPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	fmt.Printf("Pinned test tools ready: %s (%s)\n", result.Version, packages[0].Artifacts[0].SHA256)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
