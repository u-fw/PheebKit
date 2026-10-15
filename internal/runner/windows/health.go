package windows

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"kompeg/internal/core"
	"kompeg/internal/ffmpegpolicy"
	"kompeg/internal/sourcefile"
)

// Each independently sought window includes the decoder's reference preroll.
// Failure therefore identifies a conservative unsafe window, not exact bad frames.
func (r *Runner) ScanHealth(ctx context.Context, media core.Media, windowSeconds int, emit func(core.MediaHealth)) (core.MediaHealth, error) {
	report := core.MediaHealth{URI: media.Source.URI, SourceFingerprint: media.SourceFingerprint,
		DurationSeconds: media.DurationSeconds, WindowSeconds: float64(windowSeconds), State: "scanning", DamagedRanges: []core.TimeRange{}}
	if windowSeconds < 1 || windowSeconds > 30 {
		return report, fmt.Errorf("扫描窗口需要在 1–30 秒之间")
	}
	if media.DurationSeconds <= 0 || math.IsNaN(media.DurationSeconds) || math.IsInf(media.DurationSeconds, 0) || media.DurationSeconds > 21600 {
		return report, fmt.Errorf("恢复扫描需要已知时长且不超过 6 小时的视频")
	}
	if math.Ceil(media.DurationSeconds/float64(windowSeconds)) > 4320 {
		return report, fmt.Errorf("扫描超过 4320 个窗口，请增大窗口或先分段处理")
	}
	if err := localFile(media.Source, true); err != nil {
		return report, err
	}
	if media.SourceFingerprint == "" {
		return report, fmt.Errorf("请重新解析媒体后扫描")
	}
	if err := sourcefile.Verify(media.Source.URI, media.SourceFingerprint); err != nil {
		return report, err
	}
	video, audio := 0, 0
	for _, stream := range media.Streams {
		if stream.CodecType == "video" && !stream.AttachedPicture && !stream.TimedThumbnails {
			video++
		}
		if stream.CodecType == "audio" {
			audio++
		}
	}
	if video != 1 || audio > 4 {
		return report, fmt.Errorf("恢复扫描支持一条视频和最多四条音轨")
	}
	for start := 0.0; start < media.DurationSeconds; start += report.WindowSeconds {
		end := math.Min(start+report.WindowSeconds, media.DurationSeconds)
		windowCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		bad, err := r.decodeHealthRange(windowCtx, media.Source, start, end-start, true)
		cancel()
		if err != nil {
			return report, err
		}
		if bad {
			n := len(report.DamagedRanges)
			if n > 0 && report.DamagedRanges[n-1].EndSeconds == start {
				report.DamagedRanges[n-1].EndSeconds = end
			} else {
				if n >= 128 {
					return report, fmt.Errorf("失败区间超过 128 个，请先分段处理")
				}
				report.DamagedRanges = append(report.DamagedRanges, core.TimeRange{StartSeconds: start, EndSeconds: end})
			}
		}
		report.Percent = end / media.DurationSeconds * 100
		if emit != nil {
			snapshot := report
			snapshot.DamagedRanges = append([]core.TimeRange{}, report.DamagedRanges...)
			emit(snapshot)
		}
	}
	if err := sourcefile.Verify(media.Source.URI, media.SourceFingerprint); err != nil {
		return report, err
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	report.State = "completed"
	return report, nil
}

// Non-media failures must abort scanning instead of turning into deletion advice.
func mediaDecodeFailure(log string) bool {
	l := strings.ToLower(log)
	for _, s := range []string{"not found", "unknown ", "unsupported", "not supported", "permission denied", "no such file", "cannot allocate", "unrecognized option", "not available", "function not implemented"} {
		if strings.Contains(l, s) {
			return false
		}
	}
	for _, s := range []string{"corrupt decoded frame", "corrupt input packet", "packet corrupt", "corrupt frame", "invalid data found when processing input", "error while decoding", "error submitting packet", "error sending packet", "decoding error", "decode_slice_header", "missing picture in access unit", "error parsing", "invalid nal unit", "concealing ", "error decoding"} {
		if strings.Contains(l, s) {
			return true
		}
	}
	return false
}

func (r *Runner) decodeHealthRange(ctx context.Context, source core.Source, start, duration float64, seek bool) (bool, error) {
	args := []string{"-hide_banner", "-nostdin", "-loglevel", "error", "-xerror", "-max_error_rate", "0", "-progress", "pipe:1", "-stats_period", "30", "-nostats", "-threads", "1", "-err_detect", "explode"}
	if seek {
		args = append(args, "-ss", strconv.FormatFloat(start, 'f', -1, 64), "-accurate_seek")
	}
	args = append(args, ffmpegpolicy.LocalInput(source.URI)...)
	if seek {
		args = append(args, "-t", strconv.FormatFloat(duration, 'f', -1, 64))
	}
	args = append(args, "-map", "0:V:0", "-map", "0:a?", "-sn", "-dn", "-f", "null", "-")
	cmd := exec.CommandContext(ctx, r.FFmpeg, args...)
	hideProcess(cmd)
	stdout, stderr := &boundedBuffer{limit: 1024 * 1024}, &boundedBuffer{limit: 64 * 1024}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if stdout.overflow || stderr.overflow {
		return false, fmt.Errorf("解码扫描输出超过安全上限")
	}
	if err != nil || strings.TrimSpace(stderr.String()) != "" {
		if mediaDecodeFailure(stderr.String()) {
			return true, nil
		}
		return false, fmt.Errorf("解码检查未能完成（不会建议删除）：%v %s", err, strings.TrimSpace(stderr.String()))
	}
	var progress core.Progress
	if err := readProgress(bytes.NewReader(stdout.Bytes()), duration, func(p core.Progress) { progress = p }); err != nil {
		return false, err
	}
	// Missing/truncated tails can otherwise report successful process exit.
	return progress.Frame == 0 || !progress.Finished || progress.CurrentTimeSeconds < duration-math.Min(.25, duration/2), nil
}
