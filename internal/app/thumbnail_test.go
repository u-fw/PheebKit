package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"kompeg/internal/core"
)

func TestThumbnailConcurrencyAndCancellation(t *testing.T) {
	gate := make(chan struct{}, 2)
	first, err := acquireThumbnailSlot(context.Background(), gate)
	if err != nil {
		t.Fatal(err)
	}
	second, err := acquireThumbnailSlot(context.Background(), gate)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if release, err := acquireThumbnailSlot(ctx, gate); err == nil {
		release()
		t.Fatal("waiting caller ignored cancellation")
	}
	first()
	second()
	var active, maximum atomic.Int32
	var workers sync.WaitGroup
	for index := 0; index < 12; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			release, err := acquireThumbnailSlot(context.Background(), gate)
			if err != nil {
				t.Error(err)
				return
			}
			count := active.Add(1)
			for previous := maximum.Load(); count > previous; previous = maximum.Load() {
				if maximum.CompareAndSwap(previous, count) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			active.Add(-1)
			release()
		}()
	}
	workers.Wait()
	if maximum.Load() > 2 || active.Load() != 0 || len(gate) != 0 {
		t.Fatalf("thumbnail extraction was unbounded: maximum=%d active=%d slots=%d", maximum.Load(), active.Load(), len(gate))
	}
}

func TestThumbnailStreamPrefersEmbeddedCover(t *testing.T) {
	media := core.Media{Streams: []core.Stream{{Index: 0, CodecType: "video", TimedThumbnails: true}, {Index: 1, CodecType: "video"}, {Index: 3, CodecType: "video", AttachedPicture: true}}}
	stream, origin, found := thumbnailStream(media)
	if !found || stream.Index != 3 || origin != "embedded-cover" {
		t.Fatalf("wrong thumbnail stream: %+v %s", stream, origin)
	}
	media.Streams = media.Streams[:2]
	stream, origin, found = thumbnailStream(media)
	if !found || stream.Index != 1 || origin != "video-frame" {
		t.Fatalf("timed thumbnail used as video frame: %+v %s", stream, origin)
	}
	media.Streams = media.Streams[:1]
	if _, _, found := thumbnailStream(media); found {
		t.Fatal("timed thumbnail used as motion video")
	}
}

func TestThumbnailRejectsInvalidAndOversizedData(t *testing.T) {
	if _, err := thumbnailData([]byte("not an image"), "video-frame"); err == nil {
		t.Fatal("invalid image accepted")
	}
	if _, err := thumbnailData(make([]byte, thumbnailLimit+1), "video-frame"); err == nil {
		t.Fatal("oversized data accepted")
	}
	image := image.NewRGBA(image.Rect(0, 0, 1025, 1))
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, image, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := thumbnailData(buffer.Bytes(), "video-frame"); err == nil {
		t.Fatal("oversized dimensions accepted")
	}
}

func TestRealFFmpegThumbnailAndCover(t *testing.T) {
	if os.Getenv("KOMPEG_INTEGRATION") != "1" {
		t.Skip("set KOMPEG_INTEGRATION=1 to run real FFmpeg integration")
	}
	environment := discoverEnvironment(binaryConfig{})
	directory := t.TempDir()
	video := filepath.Join(directory, "视频 frame.mp4")
	cover := filepath.Join(directory, "封面.jpg")
	withCover := filepath.Join(directory, "视频 cover.mp4")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, environment.FFmpeg.Path, "-hide_banner", "-loglevel", "error", "-nostdin", "-f", "lavfi", "-i", "color=c=blue:s=320x180:r=10", "-t", "1", "-c:v", "libx264", "-pix_fmt", "yuv420p", video)
	hidePlatformProcess(command)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("create fixture: %v %s", err, output)
	}
	media := core.Media{Source: source(video), DurationSeconds: 1, Streams: []core.Stream{{Index: 0, CodecType: "video"}}}
	thumbnail, err := ffmpegThumbnail(ctx, environment.FFmpeg.Path, media)
	if err != nil {
		t.Fatal(err)
	}
	if thumbnail.Source != "video-frame" || thumbnail.Width != 512 || thumbnail.Height > 512 {
		t.Fatalf("invalid real video thumbnail: %+v", thumbnail)
	}
	assertThumbnailColour(t, thumbnail, false)
	coverImage := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			coverImage.Set(x, y, color.RGBA{R: 230, A: 255})
		}
	}
	var coverData bytes.Buffer
	if err := jpeg.Encode(&coverData, coverImage, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cover, coverData.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	command = exec.CommandContext(ctx, environment.FFmpeg.Path, "-hide_banner", "-loglevel", "error", "-nostdin", "-i", video, "-i", cover, "-map", "0", "-map", "1", "-c", "copy", "-disposition:v:1", "attached_pic", withCover)
	hidePlatformProcess(command)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("attach cover: %v %s", err, output)
	}
	service := NewService()
	defer service.Close()
	media, err = service.Probe(withCover)
	if err != nil {
		t.Fatal(err)
	}
	thumbnail, err = ffmpegThumbnail(ctx, environment.FFmpeg.Path, media)
	if err != nil {
		t.Fatal(err)
	}
	if thumbnail.Source != "embedded-cover" {
		t.Fatalf("embedded cover ignored: %s", thumbnail.Source)
	}
	assertThumbnailColour(t, thumbnail, true)
	actual, err := service.GetThumbnail(withCover)
	if err != nil {
		t.Fatal(err)
	}
	if actual.Source != "windows-shell" && actual.Source != "embedded-cover" {
		t.Fatalf("unexpected thumbnail source: %s", actual.Source)
	}
	t.Logf("real thumbnail: source=%s dimensions=%dx%d", actual.Source, actual.Width, actual.Height)
}

func assertThumbnailColour(t *testing.T, thumbnail Thumbnail, red bool) {
	t.Helper()
	encoded := strings.SplitN(thumbnail.DataURL, ",", 2)
	if len(encoded) != 2 {
		t.Fatal("no data URI")
	}
	data, err := base64.StdEncoding.DecodeString(encoded[1])
	if err != nil {
		t.Fatal(err)
	}
	frame, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	r, _, b, _ := frame.At(thumbnail.Width/2, thumbnail.Height/2).RGBA()
	if red && r <= b {
		t.Fatal("embedded red cover was not selected")
	}
	if !red && b <= r {
		t.Fatal("blue video frame was not selected")
	}
}
