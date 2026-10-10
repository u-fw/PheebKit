package app

import (
	"context"
	"io"
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
	service := NewService()
	defer service.Close()
	service.runner.replace(runner)
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

func TestRealSharedConfigurationStaysConsistentWithEachInstance(t *testing.T) {
	if os.Getenv("KOMPEG_INTEGRATION") != "1" {
		t.Skip("set KOMPEG_INTEGRATION=1 to test shared real FFmpeg configuration")
	}
	directory := t.TempDir()
	t.Setenv("APPDATA", directory)
	t.Setenv("XDG_CONFIG_HOME", directory)
	for _, name := range []string{"KOMPEG_FFMPEG", "KOMPEG_FFPROBE", "FFMPEG_PATH", "FFPROBE_PATH"} {
		t.Setenv(name, "")
	}
	first, second := NewService(), NewService()
	defer first.Close()
	defer second.Close()
	before := second.GetEnvironment()
	if !before.Ready {
		t.Fatalf("real binaries unavailable: %s", before.Message)
	}
	tools := filepath.Join(directory, "tools")
	if err := os.Mkdir(tools, 0700); err != nil {
		t.Fatal(err)
	}
	ffmpeg, ffprobe := filepath.Join(tools, "ffmpeg.exe"), filepath.Join(tools, "ffprobe.exe")
	copyTestBinary(t, before.FFmpeg.Path, ffmpeg)
	copyTestBinary(t, before.FFprobe.Path, ffprobe)
	if _, err := first.ConfigureFFmpeg(ffmpeg, ffprobe); err != nil {
		t.Fatal(err)
	}
	foreign := second.GetEnvironment()
	if !foreign.Ready || foreign.FFmpeg.Path != before.FFmpeg.Path || previewBinary(second.runner.snapshot()) != before.FFmpeg.Path || !strings.Contains(foreign.Message, "其他窗口") {
		t.Fatalf("second instance reported binaries it was not using: %+v", foreign)
	}
	// Keep an actual pending record while retaining the original adapter under
	// a blocking test wrapper. Refresh must not switch it beneath that record.
	blocking := &environmentBlockingRunner{Runner: second.runner.snapshot(), started: make(chan struct{})}
	second.runner.replace(blocking)
	job, err := second.jobs.Enqueue(core.ExecutionPlan{JobID: "configuration-in-flight"})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-blocking.started:
	case <-time.After(2 * time.Second):
		t.Fatal("blocking job did not start")
	}
	if _, err := second.RefreshCapabilities(); err == nil || !strings.Contains(err.Error(), "等待所有任务") {
		t.Fatalf("refresh replaced an active adapter: %v", err)
	}
	if second.runner.snapshot() != blocking {
		t.Fatal("failed refresh still replaced the runner")
	}
	if err := second.Cancel(job.ID); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(2 * time.Second)
	for {
		current, _ := second.jobs.Get(job.ID)
		if current.State == core.JobCancelled {
			break
		}
		select {
		case <-deadline:
			t.Fatal("blocking job did not cancel")
		case <-time.After(10 * time.Millisecond):
		}
	}
	if _, err := second.RefreshCapabilities(); err != nil {
		t.Fatal(err)
	}
	after := second.GetEnvironment()
	if !after.Ready || after.FFmpeg.Path != ffmpeg || after.FFprobe.Path != ffprobe || previewBinary(second.runner.snapshot()) != ffmpeg || after.Message != "" {
		t.Fatalf("idle rescan did not synchronize paths: %+v", after)
	}
	// Invalid external settings are never adopted or presented as the active
	// executable; the previously valid runner remains available.
	if err := writeBinaryConfig(binaryConfig{FFmpeg: filepath.Join(tools, "missing.exe"), FFprobe: ffprobe}); err != nil {
		t.Fatal(err)
	}
	if _, err := second.RefreshCapabilities(); err == nil {
		t.Fatal("invalid external configuration was adopted")
	}
	retained := second.GetEnvironment()
	if !retained.Ready || retained.FFmpeg.Path != ffmpeg || previewBinary(second.runner.snapshot()) != ffmpeg || !strings.Contains(retained.Message, "继续使用") {
		t.Fatalf("invalid shared settings changed actual status: %+v", retained)
	}
}

func copyTestBinary(t *testing.T, source, destination string) {
	t.Helper()
	input, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
}

type environmentBlockingRunner struct {
	core.Runner
	started chan struct{}
}

func (r *environmentBlockingRunner) Start(ctx context.Context, _ core.ExecutionPlan, _ core.Callbacks) (core.Result, error) {
	close(r.started)
	<-ctx.Done()
	return core.Result{}, ctx.Err()
}

func TestFailedBinaryCheckClearsPreviousAvailability(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	binary, err := checkBinary(ctx, BinaryStatus{Path: filepath.Join(t.TempDir(), "removed.exe"), Available: true, Version: "old version"}, "ffmpeg")
	if err == nil || binary.Available || binary.Version != "" {
		t.Fatalf("removed binary kept stale availability: %+v %v", binary, err)
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
