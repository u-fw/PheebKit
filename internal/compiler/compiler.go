// Package compiler validates project intent and produces argument vectors.
package compiler

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"kompeg/internal/capabilities"
	"kompeg/internal/core"
	"kompeg/internal/ffmpegpolicy"
)

type encoderPolicy struct {
	Presets map[string]string
	Option  string
}

var encoderPolicies = map[string]encoderPolicy{
	"libx264":    {Option: "-preset", Presets: map[string]string{"ultrafast": "ultrafast", "superfast": "superfast", "veryfast": "veryfast", "faster": "faster", "fast": "fast", "medium": "medium", "slow": "slow", "slower": "slower", "veryslow": "veryslow"}},
	"libx265":    {Option: "-preset", Presets: map[string]string{"ultrafast": "ultrafast", "superfast": "superfast", "veryfast": "veryfast", "faster": "faster", "fast": "fast", "medium": "medium", "slow": "slow", "slower": "slower", "veryslow": "veryslow"}},
	"libsvtav1":  {Option: "-preset", Presets: map[string]string{"fast": "10", "medium": "8", "slow": "6"}},
	"libaom-av1": {Option: "-cpu-used", Presets: map[string]string{"fast": "8", "medium": "6", "slow": "4"}},
}

func Compile(project core.Project, media core.Media, caps core.Capabilities) (core.ExecutionPlan, error) {
	if project.SchemaVersion != core.ProjectSchemaVersion {
		return core.ExecutionPlan{}, fmt.Errorf("unsupported project schema version %d", project.SchemaVersion)
	}
	if len(project.Inputs) != 1 {
		return core.ExecutionPlan{}, fmt.Errorf("each processing task requires exactly one input")
	}
	input, output := project.Inputs[0], project.Output
	if err := validatePaths(input, output); err != nil {
		return core.ExecutionPlan{}, err
	}
	if !samePath(input.URI, media.Source.URI) {
		return core.ExecutionPlan{}, fmt.Errorf("media probe does not belong to the selected input")
	}
	settings := project.Settings
	if err := ValidateSettings(settings); err != nil {
		return core.ExecutionPlan{}, err
	}
	audioStreams, err := selectTracks(media, settings.AudioSelection, "audio", true)
	if err != nil {
		return core.ExecutionPlan{}, err
	}
	subtitleStreams, err := selectTracks(media, settings.SubtitleSelection, "subtitle", false)
	if err != nil {
		return core.ExecutionPlan{}, err
	}
	if settings.Audio == "none" {
		audioStreams = nil
	}
	if settings.Operation != "" && settings.Operation != "compress" {
		return compileUtility(project, media, caps, audioStreams, subtitleStreams)
	}
	var video *core.Stream
	audio := len(audioStreams) > 0
	for i := range media.Streams {
		if media.Streams[i].CodecType == "video" && !media.Streams[i].AttachedPicture && !media.Streams[i].TimedThumbnails && video == nil {
			video = &media.Streams[i]
		}
	}
	if video == nil {
		return core.ExecutionPlan{}, fmt.Errorf("selected media has no motion video stream (cover art and thumbnails cannot be compressed as a movie)")
	}
	duration, err := outputDuration(settings.Trim, media.DurationSeconds)
	if err != nil {
		return core.ExecutionPlan{}, err
	}
	if settings.Trim != nil && video.FrameRate > 0 && !math.IsNaN(video.FrameRate) && !math.IsInf(video.FrameRate, 0) && duration < 1/video.FrameRate {
		return core.ExecutionPlan{}, fmt.Errorf("trim duration must include at least one video frame (%.3f seconds)", 1/video.FrameRate)
	}
	if settings.Trim != nil && audio && settings.Audio == "copy" {
		return core.ExecutionPlan{}, fmt.Errorf("trimming requires AAC/Opus audio encoding for accurate boundaries; copied audio is packet-aligned")
	}
	selected, pixelFormat, encoderReason, err := capabilities.SelectVideoEncoder(caps, settings.Codec, settings.Encoder, video.BitDepth > 8 && settings.Codec != "h264")
	if err != nil {
		return core.ExecutionPlan{}, err
	}
	encoder := selected.Name
	if video.HDR != "" {
		return core.ExecutionPlan{}, fmt.Errorf("HDR compression needs a verified tone-mapping/metadata preservation policy; this spike supports SDR encoding")
	}
	if settings.Resolution < 0 || settings.Resolution > 16384 || settings.Resolution%2 != 0 {
		return core.ExecutionPlan{}, fmt.Errorf("resolution must be 0 (original) or an even height between 2 and 16384")
	}
	ext := strings.ToLower(filepath.Ext(output.URI))
	if ext != ".mp4" && ext != ".mkv" && ext != ".webm" {
		return core.ExecutionPlan{}, fmt.Errorf("output must use .mp4, .mkv or .webm")
	}
	if ext == ".webm" && (settings.Codec != "av1" || (audio && settings.Audio != "opus" && settings.Audio != "copy")) {
		return core.ExecutionPlan{}, fmt.Errorf("WebM compression requires AV1 and encoded Opus or compatible copied audio")
	}
	audioArgs, audioKbps, err := compileAudio(settings.Audio, audioStreams, ext, caps.Encoders)
	if err != nil {
		return core.ExecutionPlan{}, err
	}
	subtitleArgs, err := compileSubtitles(subtitleStreams, ext, false, caps.Encoders)
	if err != nil {
		return core.ExecutionPlan{}, err
	}
	if settings.VolumeDB != 0 && (!audio || settings.Audio == "copy") {
		return core.ExecutionPlan{}, fmt.Errorf("volume changes require a selected audio track and AAC/Opus encoding")
	}
	args := []string{"-hide_banner", "-nostdin", "-n", "-progress", "pipe:1", "-nostats"}
	if settings.Trim != nil {
		// Input seeking decodes/discards the prefix up to the requested boundary.
		// Both video and audio are encoded; output -t limits the selected duration.
		args = append(args, "-ss", strconv.FormatFloat(settings.Trim.StartSeconds, 'f', -1, 64), "-accurate_seek")
	}
	args = append(args, ffmpegpolicy.LocalInput(input.URI)...)
	if settings.Trim != nil {
		args = append(args, "-t", strconv.FormatFloat(duration, 'f', -1, 64), "-map_chapters", "-1")
	}
	args = append(args, "-map", "0:"+strconv.Itoa(video.Index))
	if settings.AudioSelection == nil && settings.Audio != "none" {
		args = append(args, "-map", "0:a?")
	} else {
		args = append(args, trackMaps(audioStreams)...)
	}
	args = append(args, trackMaps(subtitleStreams)...)
	if len(subtitleStreams) == 0 {
		args = append(args, "-sn")
	}
	args = append(args, "-dn")
	bitrate := settings.BitrateKbps
	switch settings.Mode {
	case "quality":
	case "bitrate", "size":
		if settings.Mode == "size" {
			if duration <= 0 || math.IsNaN(duration) || math.IsInf(duration, 0) || math.IsNaN(settings.TargetSizeMB) || math.IsInf(settings.TargetSizeMB, 0) || settings.TargetSizeMB <= 0 {
				return core.ExecutionPlan{}, fmt.Errorf("target size requires a positive size and known media duration")
			}
			if audio && settings.Audio == "copy" {
				return core.ExecutionPlan{}, fmt.Errorf("target size requires AAC/Opus so the audio budget is known")
			}
			// A one-pass estimate with 2%% muxing allowance, not a size guarantee.
			bitrate = int(settings.TargetSizeMB*1_000_000*8*0.98/duration/1000) - audioKbps
		}
		if bitrate < 50 || bitrate > 1_000_000 {
			return core.ExecutionPlan{}, fmt.Errorf("video bitrate must be between 50 and 1000000 Kbps after the audio budget")
		}
	default:
		return core.ExecutionPlan{}, fmt.Errorf("unsupported compression mode %q", settings.Mode)
	}
	videoArgs, err := VideoArguments(encoder, settings, bitrate)
	if err != nil {
		return core.ExecutionPlan{}, err
	}
	args = append(args, videoArgs...)
	filter, err := videoFilter(settings, *video, pixelFormat)
	if err != nil {
		return core.ExecutionPlan{}, err
	}
	// Require the requested depth on software encoders too. Custom FFmpeg
	// builds can have an 8-bit-only libx265/libx264; silently choosing another
	// format would otherwise discard depth when auto falls back to CPU.
	args = append(args, "-vf", filter, "-pix_fmt", "+"+pixelFormat)
	if hasTransform(settings.Transform) {
		args = append(args, "-metadata:s:v:0", "rotate=0")
	}
	args = append(args, audioArgs...)
	args = append(args, subtitleArgs...)
	if settings.VolumeDB != 0 {
		args = append(args, "-af", "volume="+strconv.FormatFloat(settings.VolumeDB, 'f', -1, 64)+"dB")
	}
	if ext == ".mp4" {
		args = append(args, mp4TrackTitles(audioStreams, subtitleStreams)...)
		args = append(args, "-movflags", "+faststart")
		if settings.Codec == "hevc" {
			args = append(args, "-tag:v", "hvc1")
		}
	}
	if err := validateRaw(settings.RawArguments); err != nil {
		return core.ExecutionPlan{}, err
	}
	if err := validateEncoderRaw(settings.RawArguments, encoder); err != nil {
		return core.ExecutionPlan{}, err
	}
	args = append(args, settings.RawArguments...)
	args = append(args, output.URI)
	var trim *core.TimeRange
	if settings.Trim != nil {
		copy := *settings.Trim
		trim = &copy
	}
	return core.ExecutionPlan{ProjectID: project.ID, Input: input, Output: output, Args: args,
		Preview: Preview("ffmpeg.exe", args), DurationSeconds: duration, InputSizeBytes: media.SizeBytes,
		SourceFingerprint: media.SourceFingerprint,
		Trim:              trim,
		Encoder:           encoder, EncoderKind: selected.Kind, EncoderLabel: selected.Label, EncoderReason: encoderReason}, nil
}

func compileAudio(mode string, streams []core.Stream, ext string, encoders []core.Encoder) ([]string, int, error) {
	if mode != "copy" && mode != "aac" && mode != "opus" && mode != "none" {
		return nil, 0, fmt.Errorf("unsupported audio mode %q", mode)
	}
	if len(streams) == 0 || mode == "none" {
		return []string{"-an"}, 0, nil
	}
	switch mode {
	case "copy":
		if ext == ".mp4" {
			compatible := map[string]bool{"aac": true, "mp3": true, "ac3": true, "eac3": true, "alac": true}
			for _, stream := range streams {
				if !compatible[stream.CodecName] {
					return nil, 0, fmt.Errorf("audio codec %s cannot be copied to MP4 in this spike; select AAC or Matroska", stream.CodecName)
				}
			}
		}
		if ext == ".webm" {
			for _, stream := range streams {
				if stream.CodecName != "opus" && stream.CodecName != "vorbis" {
					return nil, 0, fmt.Errorf("WebM audio copy requires Opus or Vorbis")
				}
			}
		}
		return []string{"-c:a", "copy"}, 0, nil
	case "aac":
		if !capabilities.HasEncoder(encoders, "aac") {
			return nil, 0, fmt.Errorf("AAC encoder unavailable in this FFmpeg build")
		}
		return []string{"-c:a", "aac", "-b:a", "192k"}, 192 * len(streams), nil
	case "opus":
		if !capabilities.HasEncoder(encoders, "libopus") {
			return nil, 0, fmt.Errorf("libopus encoder unavailable in this FFmpeg build")
		}
		return []string{"-c:a", "libopus", "-b:a", "128k"}, 128 * len(streams), nil
	}
	return nil, 0, fmt.Errorf("unsupported audio mode")
}

func validatePaths(input, output core.Source) error {
	if input.Kind != core.SourceLocalFile || output.Kind != core.SourceLocalFile {
		return fmt.Errorf("content URI execution requires the Android native adapter: %w", core.ErrUnsupported)
	}
	if strings.ContainsRune(input.URI, 0) || strings.ContainsRune(output.URI, 0) || !filepath.IsAbs(input.URI) || !filepath.IsAbs(output.URI) {
		return fmt.Errorf("input and output must be absolute local file paths")
	}
	if samePath(input.URI, output.URI) {
		return fmt.Errorf("input and output must be different files")
	}
	parent, err := os.Stat(filepath.Dir(output.URI))
	if err != nil {
		return fmt.Errorf("inspect output directory: %w", err)
	}
	if !parent.IsDir() {
		return fmt.Errorf("output parent must be an existing directory")
	}
	if _, err := os.Stat(output.URI); err == nil {
		return fmt.Errorf("output already exists; choose a new output path")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect output: %w", err)
	}
	return nil
}

func samePath(a, b string) bool { return strings.EqualFold(filepath.Clean(a), filepath.Clean(b)) }
