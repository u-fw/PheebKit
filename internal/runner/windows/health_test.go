package windows

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"pheebkit/internal/core"
)

func TestHealthDoesNotDeleteOnOperationalFailures(t *testing.T) {
	for _, text := range []string{"Unknown decoder 'h264'", "Decoder not found", "unsupported codec", "Permission denied", "Error submitting packet: Function not implemented", "No such file or directory", "Invalid argument", "Cannot allocate memory", "Error opening C:/corrupt-video.mp4: Invalid argument"} {
		if mediaDecodeFailure(text) {
			t.Fatalf("operational error became deletion advice: %s", text)
		}
	}
	for _, text := range []string{"corrupt decoded frame", "Error submitting packet to decoder: Invalid data found when processing input", "error while decoding MB 0 0", "Invalid NAL unit size"} {
		if !mediaDecodeFailure(text) {
			t.Fatalf("missed known corruption: %s", text)
		}
	}
}

func TestHealthScanUsesCoarsePassAndFineWindowsOnlyAfterFailure(t *testing.T) {
	type probe struct {
		start    float64
		duration float64
	}
	var probes []probe
	decode := func(start, duration float64) (bool, error) {
		probes = append(probes, probe{start: start, duration: duration})
		if duration == healthCoarseSeconds {
			return start == 0 || start == 30, nil
		}
		return start == 5 || start == 15 || start == 35, nil
	}
	var progress []float64
	ranges, err := scanHealthRanges(context.Background(), 75, 5, decode, func(percent float64, _ []core.TimeRange) {
		progress = append(progress, percent)
	})
	if err != nil {
		t.Fatal(err)
	}
	wantRanges := []core.TimeRange{
		{StartSeconds: 5, EndSeconds: 10},
		{StartSeconds: 15, EndSeconds: 20},
		{StartSeconds: 35, EndSeconds: 40},
	}
	if !reflect.DeepEqual(ranges, wantRanges) {
		t.Fatalf("ranges = %+v, want %+v", ranges, wantRanges)
	}
	wantProbes := []probe{
		{0, 30},
		{0, 5}, {5, 5}, {10, 5}, {15, 5}, {20, 5}, {25, 5},
		{30, 30},
		{30, 5}, {35, 5}, {40, 5}, {45, 5}, {50, 5}, {55, 5},
		{60, 15},
	}
	if !reflect.DeepEqual(probes, wantProbes) {
		t.Fatalf("probes = %+v\nwant %+v", probes, wantProbes)
	}
	if !reflect.DeepEqual(progress, []float64{40, 80, 100}) {
		t.Fatalf("progress = %v", progress)
	}
}

func TestHealthScanDoesNotRepeatFinePassAtMaximumWindow(t *testing.T) {
	calls := 0
	ranges, err := scanHealthRanges(context.Background(), 75, healthCoarseSeconds, func(start, duration float64) (bool, error) {
		calls++
		return start == 30, nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 || !reflect.DeepEqual(ranges, []core.TimeRange{{StartSeconds: 30, EndSeconds: 60}}) {
		t.Fatalf("calls=%d ranges=%+v", calls, ranges)
	}
}

func TestHealthScanRejectsExcessFineWindowsBeforeDecoding(t *testing.T) {
	calls := 0
	_, err := scanHealthRanges(context.Background(), healthFineWindowLimit+1, 1, func(start, duration float64) (bool, error) {
		calls++
		return false, nil
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "4320 个窗口") || calls != 0 {
		t.Fatalf("excess fine windows: calls=%d err=%v", calls, err)
	}
}

func TestHealthScanCapsTotalCoarseAndFineProbes(t *testing.T) {
	calls := 0
	ranges, err := scanHealthRanges(context.Background(), healthMaximumDuration, 5, func(float64, float64) (bool, error) {
		calls++
		return true, nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if calls != healthMaximumProbeLimit {
		t.Fatalf("decode probes = %d, want %d", calls, healthMaximumProbeLimit)
	}
	if !reflect.DeepEqual(ranges, []core.TimeRange{{StartSeconds: 0, EndSeconds: healthMaximumDuration}}) {
		t.Fatalf("ranges = %+v", ranges)
	}
}

func TestHealthScanPropagatesOperationalErrors(t *testing.T) {
	operationError := errors.New("decoder unavailable")
	_, err := scanHealthRanges(context.Background(), 30, 5, func(float64, float64) (bool, error) {
		return false, operationError
	}, nil)
	if !errors.Is(err, operationError) {
		t.Fatalf("operational error was not propagated: %v", err)
	}
}
