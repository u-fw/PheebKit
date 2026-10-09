package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"

	"kompeg/internal/core"
)

type Thumbnail struct {
	DataURL string `json:"dataUrl"`
	Source  string `json:"source"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

const thumbnailLimit = 2 * 1024 * 1024

// Bound the entire extraction pipeline, including FFmpeg fallback when Shell
// providers are busy. Waiting for a slot shares the caller's total deadline.
var thumbnailGate = make(chan struct{}, 2)

func acquireThumbnailSlot(ctx context.Context, gate chan struct{}) (func(), error) {
	select {
	case gate <- struct{}{}:
		return func() { <-gate }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type limitedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(data []byte) (int, error) {
	if len(data) > b.limit-b.Len() {
		return 0, errors.New("process output exceeds limit")
	}
	return b.Buffer.Write(data)
}

func thumbnailData(data []byte, source string) (Thumbnail, error) {
	if len(data) == 0 || len(data) > thumbnailLimit {
		return Thumbnail{}, errors.New("缩略图数据无效。")
	}
	configuration, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Thumbnail{}, err
	}
	if (format != "png" && format != "jpeg") || configuration.Width < 1 || configuration.Height < 1 || configuration.Width > 1024 || configuration.Height > 1024 {
		return Thumbnail{}, errors.New("缩略图格式或尺寸超出限制。")
	}
	return Thumbnail{DataURL: "data:image/" + format + ";base64," + base64.StdEncoding.EncodeToString(data), Source: source, Width: configuration.Width, Height: configuration.Height}, nil
}

func (s *Service) GetThumbnail(uri string) (Thumbnail, error) {
	s.mediaMu.RLock()
	media, found := s.media[uri]
	s.mediaMu.RUnlock()
	if !found {
		return Thumbnail{}, errors.New("请先导入并解析媒体，再读取封面。")
	}
	if media.Source.Kind != core.SourceLocalFile {
		return Thumbnail{}, core.ErrUnsupported
	}
	info, err := os.Stat(media.Source.URI)
	if err != nil || !info.Mode().IsRegular() {
		return Thumbnail{}, errors.New("媒体文件不存在或无法读取。")
	}
	ctx, cancel := context.WithTimeout(s.ctx, 12*time.Second)
	defer cancel()
	release, err := acquireThumbnailSlot(ctx, thumbnailGate)
	if err != nil {
		return Thumbnail{}, fmt.Errorf("等候缩略图读取已取消或超时：%w", err)
	}
	defer release()
	if data, err := platformThumbnail(ctx, media.Source.URI); err == nil {
		if thumbnail, err := thumbnailData(data, "windows-shell"); err == nil {
			return thumbnail, nil
		}
	}
	return ffmpegThumbnail(ctx, previewBinary(s.runner.snapshot()), media)
}

func thumbnailStream(media core.Media) (core.Stream, string, bool) {
	for _, stream := range media.Streams {
		if stream.CodecType == "video" && stream.AttachedPicture {
			return stream, "embedded-cover", true
		}
	}
	for _, stream := range media.Streams {
		if stream.CodecType == "video" && !stream.AttachedPicture && !stream.TimedThumbnails {
			return stream, "video-frame", true
		}
	}
	return core.Stream{}, "", false
}

func ffmpegThumbnail(ctx context.Context, binary string, media core.Media) (Thumbnail, error) {
	stream, origin, found := thumbnailStream(media)
	if !found {
		return Thumbnail{}, errors.New("该媒体没有可用封面或视频画面。")
	}
	args := []string{"-hide_banner", "-loglevel", "error", "-nostdin"}
	if origin == "video-frame" && media.DurationSeconds > 0 {
		position := min(media.DurationSeconds*0.1, 5)
		args = append(args, "-ss", strconv.FormatFloat(position, 'f', 3, 64))
	}
	args = append(args, "-i", media.Source.URI, "-map", "0:"+strconv.Itoa(stream.Index), "-frames:v", "1", "-an", "-sn", "-vf", "scale=512:512:force_original_aspect_ratio=decrease", "-c:v", "mjpeg", "-q:v", "3", "-f", "image2pipe", "pipe:1")
	command := exec.CommandContext(ctx, binary, args...)
	hidePlatformProcess(command)
	var output, stderr limitedBuffer
	output.limit, stderr.limit = thumbnailLimit, 32*1024
	command.Stdout, command.Stderr = &output, &stderr
	command.WaitDelay = time.Second
	if err := command.Run(); err != nil {
		return Thumbnail{}, fmt.Errorf("读取视频缩略图失败：%w：%s", err, stderr.String())
	}
	return thumbnailData(output.Bytes(), origin)
}

// Ensure buffers implement the bounded process-output writer contract.
var _ io.Writer = (*limitedBuffer)(nil)
