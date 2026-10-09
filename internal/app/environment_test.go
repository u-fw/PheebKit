package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"kompeg/internal/core"
)

func TestBinarySelectionPrecedence(t *testing.T) {
	values := map[string]string{"KOMPEG_FFMPEG": "explicit-ffmpeg", "FFMPEG_PATH": "alias-ffmpeg"}
	getenv := func(key string) string { return values[key] }
	if selected := selectBinary("ffmpeg", "saved-ffmpeg", getenv); selected.Source != "KOMPEG_FFMPEG" || selected.Path != "explicit-ffmpeg" {
		t.Fatalf("explicit env lost precedence: %+v", selected)
	}
	delete(values, "KOMPEG_FFMPEG")
	if selected := selectBinary("ffmpeg", "saved-ffmpeg", getenv); selected.Source != "FFMPEG_PATH" {
		t.Fatalf("alias env lost precedence: %+v", selected)
	}
	delete(values, "FFMPEG_PATH")
	if selected := selectBinary("ffmpeg", "saved-ffmpeg", getenv); selected.Source != "config" {
		t.Fatalf("saved path lost precedence: %+v", selected)
	}
	if selected := selectBinary("ffmpeg", "", getenv); selected.Source != "PATH" {
		t.Fatalf("PATH fallback missing: %+v", selected)
	}
}

func TestRefreshCapabilitiesActuallyRescans(t *testing.T) {
	runner := &testRunner{}
	service := &Service{ctx: context.Background(), runner: &runnerProxy{current: runner}}
	if _, err := service.Capabilities(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Capabilities(); err != nil {
		t.Fatal(err)
	}
	if runner.scans.Load() != 1 {
		t.Fatal("normal capability calls did not cache")
	}
	if _, err := service.RefreshCapabilities(); err != nil {
		t.Fatal(err)
	}
	if runner.scans.Load() != 2 {
		t.Fatal("refresh used stale capability cache")
	}
}

func TestConfigurationRejectsPendingJobs(t *testing.T) {
	service := NewService()
	defer service.Close()
	service.runner.replace(&testRunner{})
	if _, err := service.jobs.Enqueue(core.ExecutionPlan{JobID: "configuration-guard"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConfigureFFmpeg("unused", "unused"); err == nil || !strings.Contains(err.Error(), "等待所有任务") {
		t.Fatalf("active jobs did not prevent configuration change: %v", err)
	}
}

func TestRealConfigurationReloadWithoutRestart(t *testing.T) {
	if os.Getenv("KOMPEG_INTEGRATION") != "1" {
		t.Skip("set KOMPEG_INTEGRATION=1 to run real FFmpeg integration")
	}
	directory := t.TempDir()
	t.Setenv("APPDATA", directory)
	t.Setenv("XDG_CONFIG_HOME", directory)
	for _, name := range []string{"KOMPEG_FFMPEG", "KOMPEG_FFPROBE", "FFMPEG_PATH", "FFPROBE_PATH"} {
		t.Setenv(name, "")
	}
	service := NewService()
	defer service.Close()
	before := service.GetEnvironment()
	if !before.Ready {
		t.Fatalf("real PATH binaries unavailable: %s", before.Message)
	}
	after, err := service.ConfigureFFmpeg(before.FFmpeg.Path, "")
	if err != nil {
		t.Fatal(err)
	}
	if !after.Ready || after.FFmpeg.Source != "config" || after.FFprobe.Source != "config" {
		t.Fatalf("saved binaries not selected: %+v", after)
	}
	if binary := previewBinary(service.runner.snapshot()); binary != after.FFmpeg.Path {
		t.Fatalf("runner was not updated: %s", binary)
	}
	configuration, err := readBinaryConfig()
	if err != nil || configuration.FFprobe != after.FFprobe.Path {
		t.Fatalf("sibling ffprobe was not saved: %+v %v", configuration, err)
	}
	if _, err := service.ConfigureFFmpeg(filepath.Join(directory, "missing.exe"), ""); err == nil {
		t.Fatal("invalid binary configuration accepted")
	}
	if actual, err := readBinaryConfig(); err != nil || actual != configuration {
		t.Fatal("invalid update damaged the working configuration")
	}
	restarted := NewService()
	defer restarted.Close()
	if binary := previewBinary(restarted.runner.snapshot()); binary != after.FFmpeg.Path {
		t.Fatalf("saved configuration was not loaded: %s", binary)
	}
}

type testRunner struct{ scans atomic.Int32 }

func (*testRunner) Probe(context.Context, core.Source) (core.Media, error) {
	return core.Media{}, core.ErrUnsupported
}
func (r *testRunner) Capabilities(context.Context) (core.Capabilities, error) {
	r.scans.Add(1)
	return core.Capabilities{}, nil
}
func (*testRunner) Start(ctx context.Context, _ core.ExecutionPlan, _ core.Callbacks) (core.Result, error) {
	<-ctx.Done()
	return core.Result{}, ctx.Err()
}
func (*testRunner) Cancel(string) error { return nil }

func TestMissingExplicitEnvironmentIsActionable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	missing := filepath.Join(t.TempDir(), "missing.exe")
	environment := validateEnvironment(ctx, Environment{FFmpeg: BinaryStatus{Path: missing, Source: "KOMPEG_FFMPEG"}, FFprobe: BinaryStatus{Path: missing, Source: "PATH"}})
	if environment.Ready || environment.FFmpeg.Available {
		t.Fatal("missing binary reported ready")
	}
	if !strings.Contains(environment.Message, "KOMPEG_FFMPEG") || !strings.Contains(environment.Message, "重新启动") {
		t.Fatalf("missing explicit env is not actionable: %s", environment.Message)
	}
}

func TestDirectoryBinaryPath(t *testing.T) {
	directory := t.TempDir()
	path := binaryPath(directory, "ffmpeg")
	if filepath.Dir(path) != directory || !strings.HasPrefix(filepath.Base(path), "ffmpeg") {
		t.Fatalf("directory was not normalized: %s", path)
	}
}

func TestConfigMissingDoesNotCreateFile(t *testing.T) {
	// Change only this test process's config root; never the user's real config.
	directory := t.TempDir()
	t.Setenv("APPDATA", directory)
	t.Setenv("XDG_CONFIG_HOME", directory)
	configuration, err := readBinaryConfig()
	if err != nil || configuration != (binaryConfig{}) {
		t.Fatalf("read missing config: %+v, %v", configuration, err)
	}
	path, err := configFile()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("read-only discovery created file: %v", err)
	}
	configuration = binaryConfig{FFmpeg: "C:/tools/ffmpeg.exe", FFprobe: "C:/tools/ffprobe.exe"}
	if err := writeBinaryConfig(configuration); err != nil {
		t.Fatal(err)
	}
	if actual, err := readBinaryConfig(); err != nil || actual != configuration {
		t.Fatalf("config round-trip: %+v, %v", actual, err)
	}
	configuration.FFmpeg = "C:/updated/ffmpeg.exe"
	if err := writeBinaryConfig(configuration); err != nil {
		t.Fatal(err)
	}
	if actual, err := readBinaryConfig(); err != nil || actual != configuration {
		t.Fatalf("config replacement: %+v, %v", actual, err)
	}
}
