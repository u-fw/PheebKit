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
	if os.Getenv("KOMPEG_SETUP_INTEGRATION") != "1" {
		t.Skip("set KOMPEG_SETUP_INTEGRATION=1 for a real custom ZIP download")
	}
	directory := t.TempDir()
	t.Setenv("APPDATA", directory)
	t.Setenv("LOCALAPPDATA", directory)
	for _, variable := range []string{"KOMPEG_FFMPEG", "KOMPEG_FFPROBE", "FFMPEG_PATH", "FFPROBE_PATH"} {
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

func TestRealFFmpegSetup(t *testing.T) {
	if os.Getenv("KOMPEG_SETUP_INTEGRATION") != "1" {
		t.Skip("set KOMPEG_SETUP_INTEGRATION=1 to download and verify both real packages")
	}
	for _, source := range []string{"cn", "stable"} {
		t.Run(source, func(t *testing.T) {
			directory := t.TempDir()
			t.Setenv("APPDATA", directory)
			t.Setenv("LOCALAPPDATA", directory)
			for _, variable := range []string{"KOMPEG_FFMPEG", "KOMPEG_FFPROBE", "FFMPEG_PATH", "FFPROBE_PATH"} {
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
