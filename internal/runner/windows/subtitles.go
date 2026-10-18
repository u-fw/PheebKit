package windows

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"pheebkit/internal/core"
	"pheebkit/internal/ffmpegpolicy"
	"pheebkit/internal/sourcefile"
)

func (r *Runner) prepareSubtitle(parent context.Context, dir string, asset core.SubtitleAsset) error {
	if err := localFile(asset.Source, true); err != nil {
		return err
	}
	if asset.Fingerprint == "" {
		return fmt.Errorf("字幕准备缺少文件身份校验")
	}
	if err := sourcefile.Verify(asset.Source.URI, asset.Fingerprint); err != nil {
		return fmt.Errorf("字幕源已变化：%w", err)
	}
	if asset.StreamIndex < 0 || asset.StreamIndex > 65535 {
		return fmt.Errorf("invalid subtitle stream index")
	}
	args := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-n"}
	if asset.Format != "" {
		if asset.Format != "srt" && asset.Format != "ass" && asset.Format != "webvtt" {
			return fmt.Errorf("unsupported external subtitle demuxer")
		}
		info, err := os.Stat(asset.Source.URI)
		if err != nil {
			return err
		}
		if info.Size() > 16*1024*1024 {
			return fmt.Errorf("字幕文件超过 16 MiB")
		}
		args = append(args, "-f", asset.Format)
	}
	args = append(args, ffmpegpolicy.LocalInput(asset.Source.URI)...)
	output := filepath.Join(dir, "pheebkit-subtitles.ass")
	args = append(args, "-map", "0:"+strconv.Itoa(asset.StreamIndex), "-vn", "-an", "-dn", "-c:s", "ass", "-f", "ass", "pipe:1")
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	data, err := capture(ctx, r.FFmpeg, args, false)
	if err != nil {
		return fmt.Errorf("准备文字字幕：%w", err)
	}
	if err := sourcefile.Verify(asset.Source.URI, asset.Fingerprint); err != nil {
		return fmt.Errorf("字幕源已变化：%w", err)
	}
	if len(data) <= 0 || len(data) > 16*1024*1024 {
		return fmt.Errorf("生成的字幕文件为空或超过 16 MiB")
	}
	if !bytes.Contains(data, []byte("\nDialogue:")) {
		return fmt.Errorf("字幕文件没有可渲染的文字事件，请检查内容和字幕格式")
	}
	return os.WriteFile(output, data, 0600)
}
