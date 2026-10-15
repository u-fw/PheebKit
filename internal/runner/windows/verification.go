package windows

import (
	"context"
	"fmt"
	"math"
	"time"

	"kompeg/internal/core"
)

func (r *Runner) verifyOutput(ctx context.Context, path string, plan core.ExecutionPlan) (*core.OutputVerification, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	media, err := r.Probe(probeCtx, core.Source{Kind: core.SourceLocalFile, URI: path})
	if err != nil {
		return nil, fmt.Errorf("verify output structure: %w", err)
	}
	verification, err := verifyOutputMedia(media, plan)
	if err != nil {
		return nil, err
	}
	if plan.FullDecodeVerification {
		if math.Abs(media.DurationSeconds-plan.DurationSeconds) > math.Max(.3, plan.DurationSeconds*.001) {
			return nil, fmt.Errorf("恢复输出时长不符，拒绝保存")
		}
		decodeCtx, stopDecode := context.WithTimeout(ctx, 12*time.Hour)
		defer stopDecode()
		bad, err := r.decodeHealthRange(decodeCtx, core.Source{Kind: core.SourceLocalFile, URI: path}, 0, plan.DurationSeconds, false)
		if err != nil {
			return nil, fmt.Errorf("恢复输出完整解码校验：%w", err)
		}
		if bad {
			return nil, fmt.Errorf("恢复输出完整解码校验失败，拒绝保存")
		}
		verification.FullDecodePassed = true
	}
	return verification, nil
}

func verifyOutputMedia(media core.Media, plan core.ExecutionPlan) (*core.OutputVerification, error) {
	v := &core.OutputVerification{DurationSeconds: media.DurationSeconds, Warnings: []string{}}
	for _, stream := range media.Streams {
		if (stream.CodecType == "video" || stream.CodecType == "audio" || stream.CodecType == "subtitle") && !stream.AttachedPicture && !stream.TimedThumbnails && (stream.CodecName == "" || stream.CodecName == "unknown") {
			return nil, fmt.Errorf("verify output structure: unknown %s codec", stream.CodecType)
		}
		switch stream.CodecType {
		case "video":
			if stream.AttachedPicture || stream.TimedThumbnails {
				continue
			}
			v.VideoStreams++
			if stream.Width <= 0 || stream.Height <= 0 {
				return nil, fmt.Errorf("verify output structure: invalid video stream")
			}
			if expected := plan.ExpectedOutput; expected != nil && expected.VideoCodec != "" && stream.CodecName != expected.VideoCodec {
				return nil, fmt.Errorf("verify output structure: expected %s video, found %s", expected.VideoCodec, stream.CodecName)
			}
		case "audio":
			v.AudioStreams++
			if e := plan.ExpectedOutput; e != nil {
				if e.AudioSampleRate != 0 && stream.SampleRate != e.AudioSampleRate {
					return nil, fmt.Errorf("verify output audio: expected %d Hz, found %d", e.AudioSampleRate, stream.SampleRate)
				}
				if e.AudioChannels != 0 && stream.Channels != e.AudioChannels {
					return nil, fmt.Errorf("verify output audio: expected %d channels, found %d", e.AudioChannels, stream.Channels)
				}
				index := v.AudioStreams - 1
				if index < len(e.AudioSourceChannels) && e.AudioSourceChannels[index] > 0 && stream.Channels != e.AudioSourceChannels[index] {
					return nil, fmt.Errorf("verify output audio: source channel count changed for track %d", index+1)
				}
			}
		case "subtitle":
			v.SubtitleStreams++
		}
	}
	if v.VideoStreams+v.AudioStreams+v.SubtitleStreams == 0 {
		return nil, fmt.Errorf("verify output structure: no usable media streams")
	}
	if expected := plan.ExpectedOutput; expected != nil && (expected.VideoStreams != v.VideoStreams || expected.AudioStreams != v.AudioStreams || expected.SubtitleStreams != v.SubtitleStreams) {
		return nil, fmt.Errorf("verify output structure: expected video/audio/subtitle %d/%d/%d, found %d/%d/%d", expected.VideoStreams, expected.AudioStreams, expected.SubtitleStreams, v.VideoStreams, v.AudioStreams, v.SubtitleStreams)
	}
	// Container duration can differ from selected tracks, sparse subtitles and
	// packet boundaries. Surface differences for review rather than rejecting
	// legitimate outputs. This check does not claim every frame was decoded.
	if v.VideoStreams+v.AudioStreams > 0 {
		if media.DurationSeconds <= 0 || math.IsNaN(media.DurationSeconds) || math.IsInf(media.DurationSeconds, 0) {
			v.DurationSeconds = 0
			v.Warnings = append(v.Warnings, "输出时长无法确认，请打开文件检查。")
		} else if plan.DurationSeconds > 0 && math.Abs(media.DurationSeconds-plan.DurationSeconds) > math.Max(1, plan.DurationSeconds*0.02) {
			v.Warnings = append(v.Warnings, fmt.Sprintf("输出时长 %.2f 秒，与计划 %.2f 秒有差异，请检查所选轨道和结果。", media.DurationSeconds, plan.DurationSeconds))
		}
	}
	return v, nil
}
