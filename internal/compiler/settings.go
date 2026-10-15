package compiler

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"kompeg/internal/core"
)

// ValidateSettings checks intent independent of any particular input file.
// Compile additionally checks stream indexes, source geometry and mux compatibility.
func ValidateSettings(settings core.Settings) error {
	if err := validateEditing(settings); err != nil {
		return err
	}
	if (settings.Operation == "" || settings.Operation == "compress" || settings.Operation == "extract-audio") && (settings.Audio == "aac" || settings.Audio == "opus") && settings.AudioBitrateKbps != 0 && (settings.AudioBitrateKbps < 32 || settings.AudioBitrateKbps > 320) {
		return fmt.Errorf("audio bitrate must be 0 (default) or 32–320 Kbps")
	}
	switch settings.Operation {
	case "", "compress", "remux", "extract-audio", "extract-subtitle":
	default:
		return fmt.Errorf("unsupported operation %q", settings.Operation)
	}
	if settings.Operation == "" || settings.Operation == "compress" {
		if settings.Codec != "h264" && settings.Codec != "hevc" && settings.Codec != "av1" {
			return fmt.Errorf("unsupported video codec %q", settings.Codec)
		}
		if settings.Mode != "quality" && settings.Mode != "bitrate" && settings.Mode != "size" {
			return fmt.Errorf("unsupported compression mode %q", settings.Mode)
		}
		if settings.Audio != "copy" && settings.Audio != "aac" && settings.Audio != "opus" && settings.Audio != "none" {
			return fmt.Errorf("unsupported audio mode %q", settings.Audio)
		}
		maxQuality := 51
		if settings.Codec == "av1" {
			maxQuality = 63
		}
		if settings.Mode == "quality" && (settings.Quality < 0 || settings.Quality > maxQuality) {
			return fmt.Errorf("video quality must be between 0 and %d", maxQuality)
		}
		if settings.Mode == "bitrate" && (settings.BitrateKbps < 50 || settings.BitrateKbps > 1_000_000) {
			return fmt.Errorf("video bitrate must be between 50 and 1000000 Kbps")
		}
		if settings.Mode == "size" && (math.IsNaN(settings.TargetSizeMB) || math.IsInf(settings.TargetSizeMB, 0) || settings.TargetSizeMB <= 0) {
			return fmt.Errorf("target size must be finite and positive")
		}
		if settings.Resolution < 0 || settings.Resolution > 16384 || settings.Resolution%2 != 0 {
			return fmt.Errorf("resolution must be 0 (original) or an even height between 2 and 16384")
		}
	}
	if settings.Operation == "extract-audio" && settings.Audio != "copy" && settings.Audio != "aac" && settings.Audio != "opus" && settings.Audio != "mp3" && settings.Audio != "flac" && settings.Audio != "wav" {
		return fmt.Errorf("unsupported extraction audio format %q", settings.Audio)
	}
	if settings.Trim != nil && (math.IsNaN(settings.Trim.StartSeconds) || math.IsInf(settings.Trim.StartSeconds, 0) || math.IsNaN(settings.Trim.EndSeconds) || math.IsInf(settings.Trim.EndSeconds, 0) || settings.Trim.StartSeconds < 0 || settings.Trim.EndSeconds <= settings.Trim.StartSeconds) {
		return fmt.Errorf("trim must specify finite 0 <= start < end seconds")
	}
	if math.IsNaN(settings.VolumeDB) || math.IsInf(settings.VolumeDB, 0) || settings.VolumeDB < -60 || settings.VolumeDB > 20 {
		return fmt.Errorf("volume must be a finite value between -60 and 20 dB")
	}
	if settings.VolumeDB != 0 && (settings.Audio == "copy" || settings.Audio == "none") {
		return fmt.Errorf("volume changes require audio encoding; copied or muted audio cannot change volume")
	}
	for _, selection := range []*core.TrackSelection{settings.AudioSelection, settings.SubtitleSelection} {
		if selection == nil {
			continue
		}
		if selection.Mode != "all" && selection.Mode != "none" && selection.Mode != "selected" {
			return fmt.Errorf("track selection mode must be all, none or selected")
		}
		if selection.Mode != "selected" && len(selection.Indices) != 0 {
			return fmt.Errorf("only selected track mode accepts stream indexes")
		}
		if selection.Mode == "selected" && (len(selection.Indices) == 0 || len(selection.Indices) > 64) {
			return fmt.Errorf("select between 1 and 64 tracks")
		}
		seen := map[int]bool{}
		for _, index := range selection.Indices {
			if index < 0 || index > 65535 || seen[index] {
				return fmt.Errorf("track indexes must be unique integers between 0 and 65535")
			}
			seen[index] = true
		}
	}
	if settings.Transform != nil {
		t := settings.Transform
		if t.Rotation != 0 && t.Rotation != 90 && t.Rotation != 180 && t.Rotation != 270 {
			return fmt.Errorf("rotation must be 0, 90, 180 or 270 degrees clockwise")
		}
		if t.Crop != nil {
			for _, margin := range []int{t.Crop.Top, t.Crop.Bottom, t.Crop.Left, t.Crop.Right} {
				if margin < 0 || margin > 16384 || margin%2 != 0 {
					return fmt.Errorf("crop margins must be even pixel counts between 0 and 16384")
				}
			}
		}
	}
	if err := validateRaw(settings.RawArguments); err != nil {
		return err
	}
	for i := 0; i < len(settings.RawArguments); i += 2 {
		if len(settings.RawArguments[i+1]) > 4096 {
			return fmt.Errorf("advanced argument values must not exceed 4096 bytes")
		}
		family := map[string]string{"-x264-params": "h264", "-x265-params": "hevc", "-svtav1-params": "av1"}[settings.RawArguments[i]]
		if family != "" && family != settings.Codec {
			return fmt.Errorf("advanced option %s requires %s video encoding", settings.RawArguments[i], family)
		}
	}
	if settings.Operation != "" && settings.Operation != "compress" {
		if len(settings.RawArguments) > 0 {
			return fmt.Errorf("advanced video arguments are only supported by compression")
		}
		if hasTransform(settings.Transform) || settings.Resolution != 0 {
			return fmt.Errorf("video transforms and resolution changes require compression")
		}
	}
	if settings.Operation == "remux" || settings.Operation == "extract-subtitle" {
		if settings.Trim != nil || settings.VolumeDB != 0 {
			return fmt.Errorf("trimming and volume changes are unavailable for this stream-copy/extraction operation")
		}
	}
	return nil
}

func hasTransform(t *core.VideoTransform) bool {
	return t != nil && (t.Rotation != 0 || t.FlipHorizontal || t.FlipVertical || (t.Crop != nil && (t.Crop.Top != 0 || t.Crop.Bottom != 0 || t.Crop.Left != 0 || t.Crop.Right != 0)))
}

func selectTracks(media core.Media, selection *core.TrackSelection, kind string, defaultAll bool) ([]core.Stream, error) {
	mode := "none"
	if defaultAll {
		mode = "all"
	}
	if selection != nil {
		mode = selection.Mode
	}
	if mode == "none" {
		return nil, nil
	}
	if mode == "all" {
		var streams []core.Stream
		for _, stream := range media.Streams {
			if stream.CodecType == kind {
				streams = append(streams, stream)
			}
		}
		return streams, nil
	}
	streams := make([]core.Stream, 0, len(selection.Indices))
	for _, index := range selection.Indices {
		matches := 0
		var stream core.Stream
		for _, candidate := range media.Streams {
			if candidate.Index == index {
				matches++
				stream = candidate
			}
		}
		if matches != 1 || stream.CodecType != kind {
			return nil, fmt.Errorf("selected %s stream index %d does not exist uniquely in this input", kind, index)
		}
		streams = append(streams, stream)
	}
	return streams, nil
}

func trackMaps(streams []core.Stream) []string {
	var args []string
	for _, stream := range streams {
		args = append(args, "-map", "0:"+strconv.Itoa(stream.Index))
	}
	return args
}

func videoFilter(settings core.Settings, video core.Stream, pixelFormat string) (string, error) {
	filters := []string{}
	width, height := video.Width, video.Height
	if video.DisplayRotation == 90 || video.DisplayRotation == 270 {
		width, height = height, width
	}
	if hasTransform(settings.Transform) {
		if width <= 0 || height <= 0 {
			return "", fmt.Errorf("video transforms require known positive source dimensions")
		}
		if video.DisplayRotation != 0 && video.DisplayRotation != 90 && video.DisplayRotation != 180 && video.DisplayRotation != 270 {
			return "", fmt.Errorf("video transforms require a supported 0/90/180/270 degree display rotation; input has %.3f degrees", video.DisplayRotation)
		}
	}
	if t := settings.Transform; t != nil {
		if c := t.Crop; c != nil && (c.Top+c.Bottom+c.Left+c.Right != 0) {
			width -= c.Left + c.Right
			height -= c.Top + c.Bottom
			if width < 2 || height < 2 || width%2 != 0 || height%2 != 0 {
				return "", fmt.Errorf("crop must leave an even video width and height of at least 2 pixels")
			}
			filters = append(filters, fmt.Sprintf("crop=w=%d:h=%d:x=%d:y=%d:exact=1", width, height, c.Left, c.Top))
		}
		switch t.Rotation {
		case 90:
			filters = append(filters, "transpose=clock")
			width, height = height, width
		case 180:
			filters = append(filters, "hflip", "vflip")
		case 270:
			filters = append(filters, "transpose=cclock")
			width, height = height, width
		}
		if t.FlipHorizontal {
			filters = append(filters, "hflip")
		}
		if t.FlipVertical {
			filters = append(filters, "vflip")
		}
	}
	filter := "scale=iw:ih"
	if settings.Resolution > 0 && settings.Resolution != height {
		filter = "scale=-2:" + strconv.Itoa(settings.Resolution)
	}
	filters = append(filters, filter, "format="+pixelFormat)
	return strings.Join(filters, ","), nil
}
