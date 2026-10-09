// Package compiler validates project intent and produces argument vectors.
package compiler

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ffmpegstudio/internal/capabilities"
	"ffmpegstudio/internal/core"
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
		return core.ExecutionPlan{}, fmt.Errorf("compression spike requires exactly one input")
	}
	input, output := project.Inputs[0], project.Output
	if err := validatePaths(input, output); err != nil {
		return core.ExecutionPlan{}, err
	}
	if !samePath(input.URI, media.Source.URI) {
		return core.ExecutionPlan{}, fmt.Errorf("media probe does not belong to the selected input")
	}
	var video *core.Stream
	audio := false
	for i := range media.Streams {
		if media.Streams[i].CodecType == "video" && !media.Streams[i].AttachedPicture && !media.Streams[i].TimedThumbnails && video == nil {
			video = &media.Streams[i]
		}
		if media.Streams[i].CodecType == "audio" {
			audio = true
		}
	}
	if video == nil {
		return core.ExecutionPlan{}, fmt.Errorf("selected media has no motion video stream (cover art and thumbnails cannot be compressed as a movie)")
	}
	settings := project.Settings
	encoder := ""
	for _, codec := range capabilities.Resolve(caps.Encoders) {
		if codec.ID == settings.Codec && codec.Available {
			encoder = codec.Encoder
		}
	}
	if encoder == "" {
		return core.ExecutionPlan{}, fmt.Errorf("no supported CPU encoder available for %q in this FFmpeg build", settings.Codec)
	}
	if video.HDR != "" {
		return core.ExecutionPlan{}, fmt.Errorf("HDR compression needs a verified tone-mapping/metadata preservation policy; this spike supports SDR encoding")
	}
	if settings.Resolution < 0 || settings.Resolution > 16384 || settings.Resolution%2 != 0 {
		return core.ExecutionPlan{}, fmt.Errorf("resolution must be 0 (original) or an even height between 2 and 16384")
	}
	policy := encoderPolicies[encoder]
	preset, ok := policy.Presets[settings.Preset]
	if !ok {
		return core.ExecutionPlan{}, fmt.Errorf("preset %q is unsupported for encoder %s", settings.Preset, encoder)
	}
	ext := strings.ToLower(filepath.Ext(output.URI))
	if ext != ".mp4" && ext != ".mkv" && ext != ".webm" {
		return core.ExecutionPlan{}, fmt.Errorf("output must use .mp4, .mkv or .webm")
	}
	if ext == ".webm" && (settings.Codec != "av1" || (audio && settings.Audio != "opus")) {
		return core.ExecutionPlan{}, fmt.Errorf("WebM compression requires AV1 and Opus in this spike")
	}
	audioArgs, audioKbps, err := compileAudio(settings.Audio, audio, media, ext, caps.Encoders)
	if err != nil {
		return core.ExecutionPlan{}, err
	}
	args := []string{"-hide_banner", "-nostdin", "-n", "-progress", "pipe:1", "-nostats", "-i", input.URI, "-map", "0:" + strconv.Itoa(video.Index), "-map", "0:a?", "-sn", "-dn", "-c:v", encoder, policy.Option, preset}
	switch settings.Mode {
	case "quality":
		maxCRF := 51
		if settings.Codec == "av1" {
			maxCRF = 63
		}
		if settings.Quality < 0 || settings.Quality > maxCRF {
			return core.ExecutionPlan{}, fmt.Errorf("CRF must be between 0 and %d", maxCRF)
		}
		args = append(args, "-crf", strconv.Itoa(settings.Quality))
		if encoder == "libaom-av1" {
			args = append(args, "-b:v", "0")
		}
	case "bitrate", "size":
		bitrate := settings.BitrateKbps
		if settings.Mode == "size" {
			if media.DurationSeconds <= 0 || math.IsNaN(settings.TargetSizeMB) || math.IsInf(settings.TargetSizeMB, 0) || settings.TargetSizeMB <= 0 {
				return core.ExecutionPlan{}, fmt.Errorf("target size requires a positive size and known media duration")
			}
			if audio && settings.Audio == "copy" {
				return core.ExecutionPlan{}, fmt.Errorf("target size requires AAC/Opus so the audio budget is known")
			}
			// A one-pass estimate with 2%% muxing allowance, not a size guarantee.
			bitrate = int(settings.TargetSizeMB*1_000_000*8*0.98/media.DurationSeconds/1000) - audioKbps
		}
		if bitrate < 50 || bitrate > 1_000_000 {
			return core.ExecutionPlan{}, fmt.Errorf("video bitrate must be between 50 and 1000000 Kbps after the audio budget")
		}
		args = append(args, "-b:v", strconv.Itoa(bitrate)+"k")
	default:
		return core.ExecutionPlan{}, fmt.Errorf("unsupported compression mode %q", settings.Mode)
	}
	if settings.Resolution > 0 && settings.Resolution != video.Height {
		args = append(args, "-vf", "scale=-2:"+strconv.Itoa(settings.Resolution))
	}
	pixelFormat := "yuv420p"
	if video.BitDepth > 8 && settings.Codec != "h264" {
		pixelFormat = "yuv420p10le"
	}
	args = append(args, "-pix_fmt", pixelFormat)
	args = append(args, audioArgs...)
	if ext == ".mp4" {
		args = append(args, "-movflags", "+faststart")
		if settings.Codec == "hevc" {
			args = append(args, "-tag:v", "hvc1")
		}
	}
	if err := validateRaw(settings.RawArguments); err != nil {
		return core.ExecutionPlan{}, err
	}
	args = append(args, settings.RawArguments...)
	args = append(args, output.URI)
	return core.ExecutionPlan{ProjectID: project.ID, Input: input, Output: output, Args: args,
		Preview: Preview("ffmpeg.exe", args), DurationSeconds: media.DurationSeconds, InputSizeBytes: media.SizeBytes}, nil
}

func compileAudio(mode string, hasAudio bool, media core.Media, ext string, encoders []core.Encoder) ([]string, int, error) {
	if mode != "copy" && mode != "aac" && mode != "opus" {
		return nil, 0, fmt.Errorf("unsupported audio mode %q", mode)
	}
	if !hasAudio {
		return []string{"-an"}, 0, nil
	}
	switch mode {
	case "copy":
		if ext == ".mp4" {
			compatible := map[string]bool{"aac": true, "mp3": true, "ac3": true, "eac3": true, "alac": true}
			for _, stream := range media.Streams {
				if stream.CodecType == "audio" && !compatible[stream.CodecName] {
					return nil, 0, fmt.Errorf("audio codec %s cannot be copied to MP4 in this spike; select AAC or Matroska", stream.CodecName)
				}
			}
		}
		return []string{"-c:a", "copy"}, 0, nil
	case "aac":
		if !capabilities.HasEncoder(encoders, "aac") {
			return nil, 0, fmt.Errorf("AAC encoder unavailable in this FFmpeg build")
		}
		return []string{"-c:a", "aac", "-b:a", "192k"}, 192 * countAudio(media), nil
	case "opus":
		if !capabilities.HasEncoder(encoders, "libopus") {
			return nil, 0, fmt.Errorf("libopus encoder unavailable in this FFmpeg build")
		}
		return []string{"-c:a", "libopus", "-b:a", "128k"}, 128 * countAudio(media), nil
	}
	return nil, 0, fmt.Errorf("unsupported audio mode")
}

func countAudio(media core.Media) int {
	count := 0
	for _, stream := range media.Streams {
		if stream.CodecType == "audio" {
			count++
		}
	}
	return count
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
	if _, err := os.Stat(output.URI); err == nil {
		return fmt.Errorf("output already exists; choose a new output path")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect output: %w", err)
	}
	return nil
}

func samePath(a, b string) bool { return strings.EqualFold(filepath.Clean(a), filepath.Clean(b)) }
