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

	"pheebkit/internal/core"
	"pheebkit/internal/ffmpegpolicy"
	"pheebkit/internal/processguard"
	"pheebkit/internal/sourcefile"
)

const (
	healthCoarseSeconds     = 30
	healthMaximumDuration   = 21600
	healthFineWindowLimit   = 4320
	healthMaximumProbeLimit = healthFineWindowLimit + healthMaximumDuration/healthCoarseSeconds
)

// Each independently sought window includes the decoder's reference preroll.
// Failure therefore identifies a conservative unsafe window, not exact bad frames.
func (r *Runner) ScanHealth(ctx context.Context, media core.Media, windowSeconds int, emit func(core.MediaHealth)) (core.MediaHealth, error) {
	report := core.MediaHealth{URI: media.Source.URI, SourceFingerprint: media.SourceFingerprint,
		DurationSeconds: media.DurationSeconds, WindowSeconds: float64(windowSeconds), State: "scanning", DamagedRanges: []core.TimeRange{}}
	if windowSeconds < 1 || windowSeconds > 30 {
		return report, fmt.Errorf("扫描窗口需要在 1–30 秒之间")
	}
	if media.DurationSeconds <= 0 || math.IsNaN(media.DurationSeconds) || math.IsInf(media.DurationSeconds, 0) || media.DurationSeconds > healthMaximumDuration {
		return report, fmt.Errorf("恢复扫描需要已知时长且不超过 6 小时的视频")
	}
	if math.Ceil(media.DurationSeconds/float64(windowSeconds)) > healthFineWindowLimit {
		return report, fmt.Errorf("扫描超过 %d 个窗口，请增大窗口或先分段处理", healthFineWindowLimit)
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
	decode := func(start, duration float64) (bool, error) {
		windowCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		bad, err := r.decodeHealthRange(windowCtx, media.Source, start, duration, true)
		cancel()
		return bad, err
	}
	ranges, err := scanHealthRanges(ctx, media.DurationSeconds, windowSeconds, decode, func(percent float64, damaged []core.TimeRange) {
		report.Percent = percent
		report.DamagedRanges = append(report.DamagedRanges[:0], damaged...)
		if emit != nil {
			snapshot := report
			snapshot.DamagedRanges = append([]core.TimeRange{}, report.DamagedRanges...)
			emit(snapshot)
		}
	})
	if err != nil {
		report.DamagedRanges = append([]core.TimeRange{}, ranges...)
		return report, err
	}
	report.DamagedRanges = ranges
	if err := sourcefile.Verify(media.Source.URI, media.SourceFingerprint); err != nil {
		return report, err
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	report.State = "completed"
	return report, nil
}

type healthDecoder func(start, duration float64) (bool, error)

func scanHealthRanges(
	ctx context.Context,
	duration float64,
	windowSeconds int,
	decode healthDecoder,
	publish func(percent float64, damaged []core.TimeRange),
) ([]core.TimeRange, error) {
	ranges := []core.TimeRange{}
	if math.Ceil(duration/float64(windowSeconds)) > healthFineWindowLimit {
		return ranges, fmt.Errorf("扫描超过 %d 个窗口，请增大窗口或先分段处理", healthFineWindowLimit)
	}
	probes := 0
	probe := func(start, length float64) (bool, error) {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if probes >= healthMaximumProbeLimit {
			return false, fmt.Errorf("扫描超过 %d 个解码检查，请增大窗口或先分段处理", healthMaximumProbeLimit)
		}
		probes++
		return decode(start, length)
	}
	addDamaged := func(start, end float64) error {
		if n := len(ranges); n > 0 && ranges[n-1].EndSeconds == start {
			ranges[n-1].EndSeconds = end
			return nil
		}
		if len(ranges) >= 128 {
			return fmt.Errorf("失败区间超过 128 个，请先分段处理")
		}
		ranges = append(ranges, core.TimeRange{StartSeconds: start, EndSeconds: end})
		return nil
	}
	for coarseStart := 0.0; coarseStart < duration; coarseStart += healthCoarseSeconds {
		coarseEnd := math.Min(coarseStart+healthCoarseSeconds, duration)
		failed, err := probe(coarseStart, coarseEnd-coarseStart)
		if err != nil {
			return ranges, err
		}
		if failed && windowSeconds != healthCoarseSeconds {
			for fineStart := coarseStart; fineStart < coarseEnd; fineStart += float64(windowSeconds) {
				fineEnd := math.Min(fineStart+float64(windowSeconds), coarseEnd)
				bad, err := probe(fineStart, fineEnd-fineStart)
				if err != nil {
					return ranges, err
				}
				if bad {
					if err := addDamaged(fineStart, fineEnd); err != nil {
						return ranges, err
					}
				}
			}
		} else if failed {
			if err := addDamaged(coarseStart, coarseEnd); err != nil {
				return ranges, err
			}
		}
		if publish != nil {
			publish(coarseEnd/duration*100, append([]core.TimeRange{}, ranges...))
		}
	}
	return ranges, nil
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
	stdout, stderr := &boundedBuffer{limit: 1024 * 1024}, &boundedBuffer{limit: 64 * 1024}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err := processguard.Run(cmd)
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
