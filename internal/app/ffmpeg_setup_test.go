package app

import (
	"os"
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
