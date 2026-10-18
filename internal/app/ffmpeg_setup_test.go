package app

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFFmpegSetupSourceValidation(t *testing.T) {
	service := NewService()
	defer service.Close()
	if _, err := service.StartFFmpegSetup("arbitrary-url"); err == nil {
		t.Fatal("arbitrary download source accepted")
	}
}

func TestExplicitOverrideBlocksDownloadsAndPathWrites(t *testing.T) {
	for _, variable := range []string{"PHEEBKIT_FFMPEG", "PHEEBKIT_FFPROBE", "FFMPEG_PATH", "FFPROBE_PATH"} {
		t.Run(variable, func(t *testing.T) {
			directory := t.TempDir()
			t.Setenv("APPDATA", directory)
			t.Setenv("LOCALAPPDATA", directory)
			for _, name := range []string{"PHEEBKIT_FFMPEG", "PHEEBKIT_FFPROBE", "FFMPEG_PATH", "FFPROBE_PATH"} {
				t.Setenv(name, "")
			}
			t.Setenv(variable, filepath.Join(directory, "missing.exe"))
			service := NewService()
			defer service.Close()
			_, builtinErr := service.StartFFmpegSetup("stable")
			_, customErr := service.StartCustomFFmpegSetup("https://example.com/build.7z", strings.Repeat("a", 64), true)
			_, configureErr := service.ConfigureFFmpeg(filepath.Join(directory, "replacement.exe"), "")
			for _, err := range []error{builtinErr, customErr, configureErr} {
				if err == nil || !strings.Contains(err.Error(), variable) {
					t.Fatalf("explicit override not reported: %v", err)
				}
			}
			if service.GetFFmpegSetupStatus().Active {
				t.Fatal("blocked request started download")
			}
			configurationPath, err := configFile()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(configurationPath); !os.IsNotExist(err) {
				t.Fatalf("blocked replacement wrote configuration: %v", err)
			}
		})
	}
}

func TestRealFFmpegSetupReadinessGuard(t *testing.T) {
	if os.Getenv("PHEEBKIT_INTEGRATION") != "1" {
		t.Skip("set PHEEBKIT_INTEGRATION=1 to verify guards with real PATH binaries")
	}
	makeIntegrationPATHAvailable(t)
	directory := t.TempDir()
	t.Setenv("APPDATA", directory)
	for _, name := range []string{"PHEEBKIT_FFMPEG", "PHEEBKIT_FFPROBE", "FFMPEG_PATH", "FFPROBE_PATH"} {
		t.Setenv(name, "")
	}
	service := NewService()
	defer service.Close()
	for _, source := range []string{"PATH", "config"} {
		environment := service.GetEnvironment()
		if !environment.Ready {
			t.Fatalf("real environment unavailable: %s", environment.Message)
		}
		if source == "config" {
			if _, err := service.ConfigureFFmpeg(environment.FFmpeg.Path, environment.FFprobe.Path); err != nil {
				t.Fatal(err)
			}
		}
		_, builtinErr := service.StartFFmpegSetup("stable")
		_, customErr := service.StartCustomFFmpegSetup("https://example.com/build.7z", strings.Repeat("a", 64), true)
		for _, err := range []error{builtinErr, customErr} {
			if err == nil || !strings.Contains(err.Error(), "无需重复下载") {
				t.Fatalf("%s readiness did not block download: %v", source, err)
			}
		}
		if service.GetFFmpegSetupStatus().Active {
			t.Fatal("ready environment started download")
		}
	}
}

func TestDownloadedArchitectureRejectsInvalidBinaries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ffmpeg.exe")
	if err := os.WriteFile(path, []byte("not a Windows executable"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateDownloadedArchitecture(path); err == nil {
		t.Fatal("invalid executable accepted")
	}
	header := make([]byte, 128)
	copy(header, "MZ")
	binary.LittleEndian.PutUint32(header[0x3c:], 64)
	copy(header[64:], "PE\x00\x00")
	binary.LittleEndian.PutUint16(header[68:], 0x14c)
	if err := os.WriteFile(path, header, 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateDownloadedArchitecture(path); err == nil || !strings.Contains(err.Error(), "不是 Windows x64") {
		t.Fatalf("wrong architecture accepted or misreported: %v", err)
	}
}

func TestRealCustomFFmpegSetup(t *testing.T) {
	if os.Getenv("PHEEBKIT_SETUP_INTEGRATION") != "1" {
		t.Skip("set PHEEBKIT_SETUP_INTEGRATION=1 for a real custom ZIP download")
	}
	directory := t.TempDir()
	t.Setenv("APPDATA", directory)
	t.Setenv("LOCALAPPDATA", directory)
	t.Setenv("PATH", "")
	for _, variable := range []string{"PHEEBKIT_FFMPEG", "PHEEBKIT_FFPROBE", "FFMPEG_PATH", "FFPROBE_PATH"} {
		t.Setenv(variable, "")
	}
	service := NewService()
	defer service.Close()
	const link = "https://www.gyan.dev/ffmpeg/builds/packages/ffmpeg-9.0.2-essentials_build.zip"
	const digest = "60f467265b1e312373dbcd92200c2618a74850f98d3d078e94296bb3fa2047ba"
	if _, err := service.StartCustomFFmpegSetup(link, digest, true); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(12 * time.Minute)
	for {
		status := service.GetFFmpegSetupStatus()
		if !status.Active {
			if status.Stage != "completed" || !strings.HasPrefix(status.Version, "9.0.2") {
				t.Fatalf("custom download failed: %+v", status)
			}
			t.Logf("custom ZIP completed: %s", status.Version)
			break
		}
		if time.Now().After(deadline) {
			service.CancelFFmpegSetup()
			t.Fatal("custom download timed out")
		}
		time.Sleep(250 * time.Millisecond)
	}
	environment := service.GetEnvironment()
	if !environment.Ready || !strings.HasPrefix(environment.FFmpeg.Path, directory) {
		t.Fatalf("custom paths not configured: %+v", environment)
	}
	if _, err := service.Capabilities(); err != nil {
		t.Fatal(err)
	}
	restarted := NewService()
	defer restarted.Close()
	if !restarted.GetEnvironment().Ready {
		t.Fatal("custom configuration not persisted")
	}
}

func TestRealCustom7zFFmpegSetup(t *testing.T) {
	if os.Getenv("PHEEBKIT_SETUP_INTEGRATION") != "1" {
		t.Skip("set PHEEBKIT_SETUP_INTEGRATION=1 for a real custom 7z download")
	}
	directory := t.TempDir()
	t.Setenv("APPDATA", directory)
	t.Setenv("LOCALAPPDATA", directory)
	t.Setenv("PATH", "")
	for _, variable := range []string{"PHEEBKIT_FFMPEG", "PHEEBKIT_FFPROBE", "FFMPEG_PATH", "FFPROBE_PATH"} {
		t.Setenv(variable, "")
	}
	service := NewService()
	defer service.Close()
	const link = "https://www.gyan.dev/ffmpeg/builds/packages/ffmpeg-9.0.2-full_build.7z"
	const digest = "f0e46253c70dfe902bac915dfb4224f0cf1b7c6eeab9da2ccc9a5581f9a71b13"
	if _, err := service.StartCustomFFmpegSetup(link, digest, true); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(12 * time.Minute)
	for {
		status := service.GetFFmpegSetupStatus()
		if !status.Active {
			if status.Stage != "completed" || !strings.HasPrefix(status.Version, "9.0.2") {
				t.Fatalf("custom download failed: %+v", status)
			}
			t.Logf("custom 7z completed: %s", status.Version)
			break
		}
		if time.Now().After(deadline) {
			service.CancelFFmpegSetup()
			t.Fatal("custom download timed out")
		}
		time.Sleep(250 * time.Millisecond)
	}
	environment := service.GetEnvironment()
	if !environment.Ready || !strings.HasPrefix(environment.FFmpeg.Path, directory) {
		t.Fatalf("custom paths not configured: %+v", environment)
	}
	if _, err := service.Capabilities(); err != nil {
		t.Fatal(err)
	}
	restarted := NewService()
	defer restarted.Close()
	if !restarted.GetEnvironment().Ready {
		t.Fatal("custom configuration not persisted")
	}
}

func TestRealFFmpegSetup(t *testing.T) {
	if os.Getenv("PHEEBKIT_SETUP_INTEGRATION") != "1" {
		t.Skip("set PHEEBKIT_SETUP_INTEGRATION=1 to download and verify both real packages")
	}
	for _, source := range []string{"cn", "stable"} {
		t.Run(source, func(t *testing.T) {
			directory := t.TempDir()
			t.Setenv("APPDATA", directory)
			t.Setenv("LOCALAPPDATA", directory)
			t.Setenv("PATH", "")
			for _, variable := range []string{"PHEEBKIT_FFMPEG", "PHEEBKIT_FFPROBE", "FFMPEG_PATH", "FFPROBE_PATH"} {
				t.Setenv(variable, "")
			}
			service := NewService()
			defer service.Close()
			if _, err := service.StartFFmpegSetup(source); err != nil {
				t.Fatal(err)
			}
			if _, err := service.StartFFmpegSetup(source); err == nil {
				t.Fatal("duplicate install accepted")
			}
			deadline := time.Now().Add(15 * time.Minute)
			lastStage := ""
			for {
				status := service.GetFFmpegSetupStatus()
				if status.Stage != lastStage {
					t.Logf("%s: %s %s %s", source, status.Stage, status.Package, status.Error)
					lastStage = status.Stage
				}
				if !status.Active {
					if status.Stage != "completed" {
						t.Fatalf("installation failed: %+v", status)
					}
					break
				}
				if time.Now().After(deadline) {
					service.CancelFFmpegSetup()
					t.Fatal("download timed out")
				}
				time.Sleep(250 * time.Millisecond)
			}
			environment := service.GetEnvironment()
			if !environment.Ready || !strings.HasPrefix(environment.FFmpeg.Path, directory) {
				t.Fatalf("private configuration not ready: %+v", environment)
			}
			caps, err := service.Capabilities()
			if err != nil {
				t.Fatal(err)
			}
			if len(caps.Encoders) == 0 {
				t.Fatal("no qualified encoders")
			}
			t.Logf("installed %s, %d qualified encoder entries", environment.FFmpeg.Version, len(caps.Encoders))
			restarted := NewService()
			defer restarted.Close()
			if !restarted.GetEnvironment().Ready {
				t.Fatal("configuration was not persisted")
			}
		})
	}
}
