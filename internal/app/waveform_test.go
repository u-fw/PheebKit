package app

import (
	"context"
	"encoding/binary"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"pheebkit/internal/core"
	"pheebkit/internal/sourcefile"
)

func TestWaveformPeaksRetainSilenceAndNegativeFullScale(t *testing.T) {
	data := make([]byte, 2048)
	binary.LittleEndian.PutUint16(data[1024:], 0x8000)
	peaks, err := waveformPeaks(data)
	if err != nil || len(peaks) != 512 || peaks[0] != 0 || peaks[256] != 1 {
		t.Fatalf("invalid peaks: %v %v", peaks, err)
	}
	for _, invalid := range [][]byte{nil, {1}, make([]byte, waveformMaxSeconds*waveformRate*2+34)} {
		if _, err := waveformPeaks(invalid); err == nil {
			t.Fatal("invalid or unbounded PCM accepted")
		}
	}
}

func TestWaveformImportAndWindowBoundary(t *testing.T) {
	s := NewService()
	defer s.Close()
	path := filepath.Join(t.TempDir(), "wave.wav")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetAudioWaveform(path, 0, 1, 1); err == nil {
		t.Fatal("unimported source accepted")
	}
	fingerprint, err := sourcefile.Fingerprint(path)
	if err != nil {
		t.Fatal(err)
	}
	s.media[path] = core.Media{Source: source(path), SourceFingerprint: fingerprint, DurationSeconds: 300, Streams: []core.Stream{{Index: 1, CodecType: "audio"}}}
	for _, window := range [][2]float64{{-1, 1}, {0, 121}, {299, 2}, {300, 1}, {0, 0}, {math.NaN(), 1}, {0, math.Inf(1)}} {
		if _, err := s.GetAudioWaveform(path, window[0], window[1], 1); err == nil {
			t.Fatal("invalid window accepted", window)
		}
	}
	if _, err := s.GetAudioWaveform(path, 0, 1, 2); err == nil {
		t.Fatal("unknown audio track accepted")
	}
	if err := os.WriteFile(path, []byte("replaced"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetAudioWaveform(path, 0, 1, 1); err == nil {
		t.Fatal("changed source accepted")
	}
}

func TestRealAudioWaveformWindowAndTrack(t *testing.T) {
	if os.Getenv("PHEEBKIT_INTEGRATION") != "1" {
		t.Skip("set PHEEBKIT_INTEGRATION=1")
	}
	s := NewService()
	defer s.Close()
	environment := discoverEnvironment(binaryConfig{})
	path := filepath.Join(t.TempDir(), "波形.wav")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, environment.FFmpeg.Path, "-hide_banner", "-loglevel", "error", "-nostdin", "-f", "lavfi", "-i", "anullsrc=r=8000:cl=mono:d=1", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000:duration=1", "-filter_complex", "[0:a][1:a]concat=n=2:v=0:a=1[a]", "-map", "[a]", "-c:a", "pcm_s16le", path)
	hidePlatformProcess(command)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v %s", err, output)
	}
	if _, err := s.Probe(path); err != nil {
		t.Fatal(err)
	}
	for _, window := range []struct{ start, peak float64 }{{0, 0}, {1, 0.12}} {
		wave, err := s.GetAudioWaveform(path, window.start, .5, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(wave.Peaks) != waveformBins || math.Abs(wave.DurationSeconds-.5) > .01 || wave.StartSeconds != window.start || wave.StreamIndex != 0 {
			t.Fatalf("wrong window: %+v", wave)
		}
		maximum := 0.0
		for _, value := range wave.Peaks {
			maximum = max(maximum, value)
		}
		if math.Abs(maximum-window.peak) > .03 {
			t.Fatalf("wrong waveform at %v: %v", window.start, maximum)
		}
	}
}
