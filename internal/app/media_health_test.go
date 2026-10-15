package app

import (
	"context"
	"testing"
	"time"

	"kompeg/internal/core"
	"kompeg/internal/jobs"
)

type healthTestRunner struct {
	testRunner
	entered chan struct{}
}

func (r *healthTestRunner) ScanHealth(ctx context.Context, m core.Media, _ int, emit func(core.MediaHealth)) (core.MediaHealth, error) {
	value := core.MediaHealth{URI: m.Source.URI, SourceFingerprint: m.SourceFingerprint, State: "scanning", Percent: 50, DamagedRanges: []core.TimeRange{{EndSeconds: 5}}}
	emit(value)
	close(r.entered)
	<-ctx.Done()
	return value, ctx.Err()
}

func TestHealthScanLifecycleAndIsolatedSnapshots(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &healthTestRunner{entered: make(chan struct{})}
	s := &Service{ctx: ctx, cancel: cancel, runner: &runnerProxy{current: r}, jobs: jobs.New(r, nil)}
	defer s.Close()
	s.cacheMedia(core.Media{Source: core.Source{URI: "C:/fixture.mp4"}, SourceFingerprint: "test"})
	if _, err := s.StartMediaHealthScan("C:/fixture.mp4", 0); err == nil {
		t.Fatal("invalid scan window admitted")
	}
	report, err := s.StartMediaHealthScan("C:/fixture.mp4", 5)
	if err != nil {
		t.Fatal(err)
	}
	<-r.entered
	if _, err := s.StartMediaHealthScan("C:/fixture.mp4", 5); err == nil {
		t.Fatal("concurrent scan admitted")
	}
	if !s.hasPendingJobs() || s.CanClose() {
		t.Fatal("active scan bypassed lifecycle")
	}
	snapshot := s.GetMediaHealthScan()
	snapshot.DamagedRanges[0].EndSeconds = 99
	if s.GetMediaHealthScan().DamagedRanges[0].EndSeconds == 99 {
		t.Fatal("bridge snapshot aliases report")
	}
	if err := s.CancelMediaHealthScan("stale"); err == nil {
		t.Fatal("stale cancellation accepted")
	}
	if err := s.CancelMediaHealthScan(report.ID); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for s.healthBusy.Load() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.GetMediaHealthScan().State != "cancelled" || s.healthBusy.Load() {
		t.Fatal("cancel not settled")
	}
	if portableSettings(core.Settings{Recovery: &core.Recovery{Mode: "delete"}}).Recovery != nil {
		t.Fatal("source recovery leaked into portable preset")
	}
	s.Close()
	if _, err := s.StartMediaHealthScan("C:/fixture.mp4", 5); err == nil {
		t.Fatal("scan admitted after close")
	}
}

func TestHealthAdmissionDoesNotHoldLifecycleWhileWaitingForConfiguration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &healthTestRunner{entered: make(chan struct{})}
	s := &Service{ctx: ctx, cancel: cancel, runner: &runnerProxy{current: r}, jobs: jobs.New(r, nil)}
	defer s.Close()
	s.configMu.Lock()
	started := make(chan struct{})
	result := make(chan error, 1)
	go func() { close(started); _, err := s.StartMediaHealthScan("C:/fixture.mp4", 5); result <- err }()
	<-started
	// Allow admission to reach its blocked configuration acquisition.
	time.Sleep(10 * time.Millisecond)
	closed := make(chan bool, 1)
	go func() { closed <- s.CanClose() }()
	select {
	case allowed := <-closed:
		if !allowed {
			s.configMu.Unlock()
			t.Fatal("idle close was blocked")
		}
	case <-time.After(time.Second):
		s.configMu.Unlock()
		t.Fatal("configuration wait held lifecycle lock")
	}
	s.configMu.Unlock()
	if err := <-result; err == nil {
		t.Fatal("scan admitted after idle close")
	}
}
