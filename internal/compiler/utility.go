package compiler

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"kompeg/internal/capabilities"
	"kompeg/internal/core"
	"kompeg/internal/ffmpegpolicy"
)

var textSubtitles = map[string]bool{"subrip": true, "srt": true, "ass": true, "ssa": true, "webvtt": true, "mov_text": true, "text": true}
var matroskaSubtitles = map[string]bool{"subrip": true, "ass": true, "ssa": true, "webvtt": true, "dvd_subtitle": true, "dvb_subtitle": true, "hdmv_pgs_subtitle": true}

// MP4 names tracks via the handler name, rather than Matroska's title tag.
func mp4TrackTitles(audio, subtitles []core.Stream) []string {
	var args []string
	for _, group := range []struct {
		kind    string
		streams []core.Stream
	}{{"a", audio}, {"s", subtitles}} {
		for i, stream := range group.streams {
			if stream.Title != "" {
				args = append(args, "-metadata:s:"+group.kind+":"+strconv.Itoa(i), "handler_name="+stream.Title)
			}
		}
	}
	return args
}

// Selected subtitle formats are checked explicitly: FFmpeg otherwise accepts an
// encoder choice and only discovers a text/bitmap or mux incompatibility later.
// FFmpeg docs: ffmpeg.html (stream selection), matroskaenc.c (native codec tags).
func compileSubtitles(streams []core.Stream, ext string, copyOnly bool, encoders []core.Encoder) ([]string, error) {
	var args []string
	for i, stream := range streams {
		encoder := "copy"
		switch ext {
		case ".mkv":
			if !matroskaSubtitles[stream.CodecName] {
				if !copyOnly && textSubtitles[stream.CodecName] {
					encoder = "subrip"
				} else {
					return nil, fmt.Errorf("subtitle codec %s cannot be copied to Matroska; extract/convert the text subtitle separately", stream.CodecName)
				}
			}
		case ".mp4":
			if stream.CodecName != "mov_text" {
				if copyOnly {
					return nil, fmt.Errorf("MP4 remux requires existing mov_text subtitles; choose Matroska or compression to convert text subtitles")
				}
				if !textSubtitles[stream.CodecName] {
					return nil, fmt.Errorf("bitmap/unsupported subtitle %s cannot be converted to MP4 text subtitles", stream.CodecName)
				}
				encoder = "mov_text"
			}
		case ".webm":
			if stream.CodecName != "webvtt" {
				if copyOnly {
					return nil, fmt.Errorf("WebM remux requires existing WebVTT subtitles")
				}
				if !textSubtitles[stream.CodecName] {
					return nil, fmt.Errorf("bitmap/unsupported subtitle %s cannot be converted to WebVTT", stream.CodecName)
				}
				encoder = "webvtt"
			}
		default:
			return nil, fmt.Errorf("unsupported subtitle container %s", ext)
		}
		if encoder != "copy" && !capabilities.HasEncoder(encoders, encoder) {
			return nil, fmt.Errorf("subtitle encoder %s unavailable in this FFmpeg build", encoder)
		}
		args = append(args, "-c:s:"+strconv.Itoa(i), encoder)
	}
	return args, nil
}

func compileUtility(project core.Project, media core.Media, caps core.Capabilities, audio, subtitles []core.Stream) (core.ExecutionPlan, error) {
	s := project.Settings
	ext := strings.ToLower(filepath.Ext(project.Output.URI))
	duration, err := outputDuration(s.Trim, media.DurationSeconds)
	if err != nil {
		return core.ExecutionPlan{}, err
	}
	args := []string{"-hide_banner", "-nostdin", "-n", "-progress", "pipe:1", "-nostats"}
	if s.Trim != nil {
		args = append(args, "-ss", strconv.FormatFloat(s.Trim.StartSeconds, 'f', -1, 64), "-accurate_seek")
	}
	args = append(args, ffmpegpolicy.LocalInput(project.Inputs[0].URI)...)
	if s.Trim != nil {
		args = append(args, "-t", strconv.FormatFloat(duration, 'f', -1, 64))
	}
	encoder, label, kind := "copy", "无需重编码", "copy"
	switch s.Operation {
	case "remux":
		if ext != ".mp4" && ext != ".mkv" && ext != ".webm" {
			return core.ExecutionPlan{}, fmt.Errorf("remux output must use .mp4, .mkv or .webm")
		}
		var video *core.Stream
		for i := range media.Streams {
			candidate := &media.Streams[i]
			if candidate.CodecType == "video" && !candidate.AttachedPicture && !candidate.TimedThumbnails {
				video = candidate
				break
			}
		}
		if video == nil {
			return core.ExecutionPlan{}, fmt.Errorf("remux requires a motion video stream; use audio extraction for audio-only media")
		}
		if ext == ".mp4" && video.CodecName != "h264" && video.CodecName != "hevc" && video.CodecName != "av1" && video.CodecName != "mpeg4" {
			return core.ExecutionPlan{}, fmt.Errorf("video codec %s is outside the supported MP4 copy policy; choose Matroska", video.CodecName)
		}
		if ext == ".webm" && video.CodecName != "vp8" && video.CodecName != "vp9" && video.CodecName != "av1" {
			return core.ExecutionPlan{}, fmt.Errorf("WebM stream copy requires VP8, VP9 or AV1 video")
		}
		audioArgs, _, err := compileAudio("copy", audio, ext, caps.Encoders)
		if err != nil {
			return core.ExecutionPlan{}, err
		}
		subtitleArgs, err := compileSubtitles(subtitles, ext, true, caps.Encoders)
		if err != nil {
			return core.ExecutionPlan{}, err
		}
		args = append(args, "-map", "0:"+strconv.Itoa(video.Index))
		args = append(args, trackMaps(audio)...)
		args = append(args, trackMaps(subtitles)...)
		args = append(args, "-dn", "-c:v", "copy")
		args = append(args, audioArgs...)
		args = append(args, subtitleArgs...)
		if len(subtitles) == 0 {
			args = append(args, "-sn")
		}
		args = append(args, "-map_metadata", "0", "-map_chapters", "0")
		if ext == ".mp4" {
			args = append(args, mp4TrackTitles(audio, subtitles)...)
			args = append(args, "-movflags", "+faststart")
			if video.CodecName == "hevc" {
				args = append(args, "-tag:v", "hvc1")
			}
		}
	case "extract-audio":
		if s.AudioSelection == nil || s.AudioSelection.Mode != "selected" || len(audio) != 1 {
			return core.ExecutionPlan{}, fmt.Errorf("audio extraction requires selecting exactly one audio track")
		}
		if len(subtitles) > 0 {
			return core.ExecutionPlan{}, fmt.Errorf("audio extraction does not output subtitles")
		}
		args = append(args, trackMaps(audio)...)
		args = append(args, "-vn", "-sn", "-dn", "-map_metadata", "0", "-map_chapters", "-1")
		audioArgs, selectedEncoder, err := extractAudioArguments(s.Audio, audio[0].CodecName, ext, caps.Encoders)
		if err != nil {
			return core.ExecutionPlan{}, err
		}
		if s.Audio == "copy" && (s.Trim != nil || s.VolumeDB != 0) {
			return core.ExecutionPlan{}, fmt.Errorf("copied audio cannot trim accurately or change volume; select audio encoding")
		}
		args = append(args, audioArgs...)
		if ext == ".m4a" {
			args = append(args, mp4TrackTitles(audio, nil)...)
		}
		encoder = selectedEncoder
		if encoder != "copy" {
			kind, label = "cpu", "音频编码"
		}
		if s.VolumeDB != 0 {
			args = append(args, "-af", "volume="+strconv.FormatFloat(s.VolumeDB, 'f', -1, 64)+"dB")
		}
	case "extract-subtitle":
		if s.SubtitleSelection == nil || s.SubtitleSelection.Mode != "selected" || len(subtitles) != 1 {
			return core.ExecutionPlan{}, fmt.Errorf("subtitle extraction requires selecting exactly one subtitle track")
		}
		if !textSubtitles[subtitles[0].CodecName] {
			return core.ExecutionPlan{}, fmt.Errorf("bitmap/unsupported subtitle %s needs OCR before it can become editable text; OCR is not provided", subtitles[0].CodecName)
		}
		encoder = map[string]string{".srt": "subrip", ".ass": "ass", ".vtt": "webvtt"}[ext]
		if encoder == "" {
			return core.ExecutionPlan{}, fmt.Errorf("text subtitle output must use .srt, .ass or .vtt")
		}
		if !capabilities.HasEncoder(caps.Encoders, encoder) {
			return core.ExecutionPlan{}, fmt.Errorf("subtitle encoder %s unavailable in this FFmpeg build", encoder)
		}
		args = append(args, trackMaps(subtitles)...)
		args = append(args, "-vn", "-an", "-dn", "-c:s", encoder, "-map_chapters", "-1")
		kind, label = "cpu", "文本字幕转换"
	default:
		return core.ExecutionPlan{}, fmt.Errorf("unsupported operation %q", s.Operation)
	}
	args = append(args, project.Output.URI)
	expected := &core.OutputExpectation{}
	switch s.Operation {
	case "remux":
		expected.VideoStreams, expected.AudioStreams, expected.SubtitleStreams = 1, len(audio), len(subtitles)
	case "extract-audio":
		expected.AudioStreams = 1
	case "extract-subtitle":
		expected.SubtitleStreams = 1
	}
	return core.ExecutionPlan{ProjectID: project.ID, Input: project.Inputs[0], Output: project.Output, Args: args, Preview: Preview("ffmpeg.exe", args), DurationSeconds: duration, InputSizeBytes: media.SizeBytes, SourceFingerprint: media.SourceFingerprint, Trim: core.CloneSettings(s).Trim, Encoder: encoder, EncoderKind: kind, EncoderLabel: label, EncoderReason: "", ExpectedOutput: expected}, nil
}

func extractAudioArguments(mode, sourceCodec, ext string, encoders []core.Encoder) ([]string, string, error) {
	if mode == "copy" {
		compatible := ext == ".mka" || (ext == ".m4a" && (sourceCodec == "aac" || sourceCodec == "alac")) || (ext == ".opus" && sourceCodec == "opus") || (ext == ".mp3" && sourceCodec == "mp3") || (ext == ".flac" && sourceCodec == "flac") || (ext == ".wav" && strings.HasPrefix(sourceCodec, "pcm_"))
		if !compatible {
			return nil, "", fmt.Errorf("audio codec %s cannot be copied to %s; choose .mka or matching audio encoding", sourceCodec, ext)
		}
		return []string{"-c:a", "copy"}, "copy", nil
	}
	encoder := map[string]string{"aac": "aac", "opus": "libopus", "mp3": "libmp3lame", "flac": "flac", "wav": "pcm_s16le"}[mode]
	requiredExt := map[string]string{"aac": ".m4a", "opus": ".opus", "mp3": ".mp3", "flac": ".flac", "wav": ".wav"}[mode]
	if encoder == "" || (ext != requiredExt && ext != ".mka") {
		return nil, "", fmt.Errorf("audio format %s requires %s (or .mka) output", mode, requiredExt)
	}
	if !capabilities.HasEncoder(encoders, encoder) {
		return nil, "", fmt.Errorf("audio encoder %s unavailable in this FFmpeg build", encoder)
	}
	args := []string{"-c:a", encoder}
	switch mode {
	case "aac":
		args = append(args, "-b:a", "192k")
	case "opus":
		args = append(args, "-b:a", "128k")
	case "mp3":
		args = append(args, "-q:a", "2")
	}
	if ext == ".m4a" {
		args = append(args, "-movflags", "+faststart")
	}
	return args, encoder, nil
}
