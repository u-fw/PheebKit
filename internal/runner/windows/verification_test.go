package windows

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"kompeg/internal/core"
)

func TestOutputVerificationRejectsStructureAndRetainsDurationWarnings(t *testing.T) {
	video := core.Stream{CodecType: "video", CodecName: "hevc", Width: 1920, Height: 1080}
	audio := core.Stream{CodecType: "audio", CodecName: "aac"}
	plan := core.ExecutionPlan{DurationSeconds: 10, ExpectedOutput: &core.OutputExpectation{VideoStreams: 1, AudioStreams: 1, VideoCodec: "hevc"}}
	media := core.Media{DurationSeconds: 10.04, Streams: []core.Stream{video, audio}}
	v, err := verifyOutputMedia(media, plan)
	if err != nil || v.VideoStreams != 1 || v.AudioStreams != 1 || len(v.Warnings) != 0 {
		t.Fatalf("valid output rejected: %+v %v", v, err)
	}
	for _, streams := range [][]core.Stream{
		{video}, {audio}, {{CodecType: "video", CodecName: "h264", Width: 1920, Height: 1080}, audio},
		{video, {CodecType: "audio", CodecName: "unknown"}},
		{{CodecType: "video", CodecName: "hevc"}, audio},
		{{CodecType: "video", CodecName: "hevc", Width: 1920, Height: 1080, AttachedPicture: true}, audio},
	} {
		if _, err := verifyOutputMedia(core.Media{DurationSeconds: 10, Streams: streams}, plan); err == nil {
			t.Fatalf("invalid output accepted: %+v", streams)
		}
	}
	media.DurationSeconds = 2
	v, err = verifyOutputMedia(media, plan)
	if err != nil || len(v.Warnings) != 1 {
		t.Fatalf("duration mismatch must be visible, not silently rejected: %+v %v", v, err)
	}
	media.DurationSeconds = 0
	v, err = verifyOutputMedia(media, plan)
	if err != nil || len(v.Warnings) != 1 {
		t.Fatal("unknown duration was not reported", err)
	}
	// Sparse text subtitles often have no container duration.
	v, err = verifyOutputMedia(core.Media{Streams: []core.Stream{{CodecType: "subtitle", CodecName: "subrip"}}}, core.ExecutionPlan{DurationSeconds: 100, ExpectedOutput: &core.OutputExpectation{SubtitleStreams: 1}})
	if err != nil || len(v.Warnings) != 0 {
		t.Fatal("sparse subtitles incorrectly failed duration checks", err)
	}
}

func TestOutputVerificationRejectsUnreadableMedia(t *testing.T) {
	runner := New("", "")
	if _, err := os.Stat(runner.FFprobe); err != nil {
		t.Skip("installed FFprobe is unavailable")
	}
	path := filepath.Join(t.TempDir(), "broken.mp4")
	if err := os.WriteFile(path, []byte("not a video"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.verifyOutput(context.Background(), path, core.ExecutionPlan{}); err == nil {
		t.Fatal("nonempty garbage passed output verification")
	}
}

func TestOutputVerificationChecksRequestedAudioFormat(t *testing.T) {
	p := core.ExecutionPlan{ExpectedOutput: &core.OutputExpectation{AudioStreams: 1, AudioSampleRate: 48000, AudioChannels: 2}}
	m := core.Media{DurationSeconds: 2, Streams: []core.Stream{{CodecType: "audio", CodecName: "aac", SampleRate: 44100, Channels: 2}}}
	if _, err := verifyOutputMedia(m, p); err == nil {
		t.Fatal("wrong rate published")
	}
	m.Streams[0].SampleRate = 48000
	m.Streams[0].Channels = 1
	if _, err := verifyOutputMedia(m, p); err == nil {
		t.Fatal("wrong channel count published")
	}
	m.Streams[0].Channels = 2
	if _, err := verifyOutputMedia(m, p); err != nil {
		t.Fatal(err)
	}
	p.ExpectedOutput.AudioChannels = 0
	p.ExpectedOutput.AudioSourceChannels = []int{6}
	if _, err := verifyOutputMedia(m, p); err == nil {
		t.Fatal("preserve channel count silently downmixed")
	}
}
