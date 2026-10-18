package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"pheebkit/internal/core"
)

func TestRealAudioProcessingThroughQueue(t *testing.T) {
	if os.Getenv("PHEEBKIT_INTEGRATION") != "1" {
		t.Skip("real FFmpeg opt-in")
	}
	t.Setenv("APPDATA", t.TempDir())
	s := NewService()
	defer s.Close()
	dir := t.TempDir()
	input := filepath.Join(dir, "audio.wav")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	env := discoverEnvironment(binaryConfig{})
	cmd := exec.CommandContext(ctx, env.FFmpeg.Path, "-v", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "sine=sample_rate=44100:duration=3", "-c:a", "pcm_s16le", input)
	hidePlatformProcess(cmd)
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, data)
	}
	m, err := s.Probe(input)
	if err != nil {
		t.Fatal(err)
	}
	s.caps = &core.Capabilities{Encoders: []core.Encoder{{Name: "pcm_s16le"}}, Filters: []string{"aresample", "afade", "equalizer"}}
	p := core.Project{SchemaVersion: 1, ID: "audio-queue", Inputs: []core.Source{m.Source}, Output: source(filepath.Join(dir, "processed.wav")), Settings: core.DefaultSettings()}
	p.Settings.Operation, p.Settings.Audio = "extract-audio", "wav"
	p.Settings.AudioSelection = &core.TrackSelection{Mode: "selected", Indices: []int{0}}
	p.Settings.AudioProcessing = &core.AudioProcessing{SampleRate: 16000, Channels: "mono", FadeInSeconds: .5, FadeOutSeconds: .5, Equalizer: []core.EqualizerBand{{FrequencyHz: 1000, Q: 1, GainDB: -3}}}
	job, err := s.Start(p)
	if err != nil {
		t.Fatal(err)
	}
	p.Settings.AudioProcessing.Equalizer[0].GainDB = 20
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		current, _ := s.jobs.Get(job.ID)
		if current.State == core.JobFailed {
			t.Fatal(current.Error)
		}
		if current.State == core.JobCompleted {
			if current.Result.Verification == nil || current.Result.Verification.AudioStreams != 1 {
				t.Fatal("queue missing verification")
			}
			out, err := s.Probe(current.Output.URI)
			if err != nil || out.Streams[0].SampleRate != 16000 || out.Streams[0].Channels != 1 {
				t.Fatal("queue ignored audio format", err)
			}
			if current.Project.Settings.AudioProcessing.Equalizer[0].GainDB != -3 {
				t.Fatal("mutable queue audio settings")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("audio queue did not settle")
}
