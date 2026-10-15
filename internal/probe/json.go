// Package probe parses FFprobe's JSON output without parsing human-readable logs.
package probe

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"kompeg/internal/core"
)

type document struct {
	Format struct {
		Filename   string `json:"filename"`
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
		Size       string `json:"size"`
		BitRate    string `json:"bit_rate"`
	} `json:"format"`
	Streams []struct {
		SampleRate       string            `json:"sample_rate"`
		Channels         int               `json:"channels"`
		ChannelLayout    string            `json:"channel_layout"`
		Index            int               `json:"index"`
		CodecType        string            `json:"codec_type"`
		CodecName        string            `json:"codec_name"`
		Width            int               `json:"width"`
		Height           int               `json:"height"`
		AverageFrameRate string            `json:"avg_frame_rate"`
		RealFrameRate    string            `json:"r_frame_rate"`
		PixelFormat      string            `json:"pix_fmt"`
		BitsPerRawSample string            `json:"bits_per_raw_sample"`
		BitsPerSample    int               `json:"bits_per_sample"`
		ColorSpace       string            `json:"color_space"`
		ColorTransfer    string            `json:"color_transfer"`
		ColorPrimaries   string            `json:"color_primaries"`
		Duration         string            `json:"duration"`
		Tags             map[string]string `json:"tags"`
		Disposition      struct {
			AttachedPicture int `json:"attached_pic"`
			TimedThumbnails int `json:"timed_thumbnails"`
		} `json:"disposition"`
		SideData []struct {
			Type     string   `json:"side_data_type"`
			Rotation *float64 `json:"rotation"`
		} `json:"side_data_list"`
	} `json:"streams"`
}

func Parse(data []byte, source core.Source) (core.Media, error) {
	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return core.Media{}, fmt.Errorf("decode FFprobe JSON: %w", err)
	}
	if len(doc.Streams) == 0 {
		return core.Media{}, fmt.Errorf("FFprobe found no media streams")
	}
	name := source.DisplayName
	if name == "" {
		name = filepath.Base(source.URI)
	}
	media := core.Media{Source: source, Name: name, Container: doc.Format.FormatName,
		SizeBytes: integer(doc.Format.Size), DurationSeconds: decimal(doc.Format.Duration),
		Bitrate: integer(doc.Format.BitRate), Streams: make([]core.Stream, 0, len(doc.Streams))}
	formatHasDuration := media.DurationSeconds > 0
	for _, raw := range doc.Streams {
		fps := rational(raw.AverageFrameRate)
		if fps == 0 {
			fps = rational(raw.RealFrameRate)
		}
		depth := int(integer(raw.BitsPerRawSample))
		if depth == 0 {
			depth = raw.BitsPerSample
		}
		if depth == 0 && raw.CodecType == "video" {
			depth = pixelDepth(raw.PixelFormat)
		}
		hdr := ""
		rotation := signedDecimal(raw.Tags["rotate"])
		switch raw.ColorTransfer {
		case "smpte2084":
			hdr = "PQ / HDR10"
		case "arib-std-b67":
			hdr = "HLG"
		}
		for _, side := range raw.SideData {
			if strings.EqualFold(side.Type, "Display Matrix") && side.Rotation != nil {
				rotation = normalizedRotation(*side.Rotation)
			}
			if strings.Contains(strings.ToLower(side.Type), "dovi") || strings.Contains(strings.ToLower(side.Type), "dolby vision") {
				hdr = "Dolby Vision"
			}
		}
		media.Streams = append(media.Streams, core.Stream{Index: raw.Index, CodecType: raw.CodecType,
			SampleRate: int(integer(raw.SampleRate)), Channels: raw.Channels, ChannelLayout: raw.ChannelLayout,
			CodecName: raw.CodecName, Width: raw.Width, Height: raw.Height, DisplayRotation: rotation, FrameRate: fps,
			PixelFormat: raw.PixelFormat, BitDepth: depth, ColorSpace: raw.ColorSpace,
			ColorTransfer: raw.ColorTransfer, ColorPrimaries: raw.ColorPrimaries, HDR: hdr,
			Language: raw.Tags["language"], Title: trackTitle(raw.Tags),
			AttachedPicture: raw.Disposition.AttachedPicture != 0, TimedThumbnails: raw.Disposition.TimedThumbnails != 0})
		if !formatHasDuration {
			media.DurationSeconds = math.Max(media.DurationSeconds, decimal(raw.Duration))
		}
	}
	return media, nil
}

func trackTitle(tags map[string]string) string {
	if title := tags["title"]; title != "" {
		return title
	}
	return tags["handler_name"]
}

func signedDecimal(value string) float64 {
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return normalizedRotation(number)
}

func normalizedRotation(rotation float64) float64 {
	if math.IsNaN(rotation) || math.IsInf(rotation, 0) {
		return 0
	}
	rotation = math.Mod(rotation, 360)
	if rotation < 0 {
		rotation += 360
	}
	quarter := math.Round(rotation/90) * 90
	if math.Abs(rotation-quarter) < 0.001 {
		rotation = math.Mod(quarter, 360)
	}
	if rotation == 0 {
		return 0
	}
	return rotation
}

func integer(s string) int64 { v, _ := strconv.ParseInt(s, 10, 64); return v }

func decimal(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(v, 0) || math.IsNaN(v) || v < 0 {
		return 0
	}
	return v
}

func rational(s string) float64 {
	parts := strings.Split(s, "/")
	if len(parts) == 2 {
		denominator := decimal(parts[1])
		if denominator > 0 {
			return decimal(parts[0]) / denominator
		}
		return 0
	}
	return decimal(s)
}

// FFprobe omits bits_per_raw_sample for many decoded video formats.
func pixelDepth(format string) int {
	for _, depth := range []string{"16", "14", "12", "10", "9"} {
		if strings.HasSuffix(format, "p"+depth+"le") || strings.HasSuffix(format, "p"+depth+"be") || strings.HasSuffix(format, "gray"+depth+"le") || strings.HasSuffix(format, "gray"+depth+"be") {
			v, _ := strconv.Atoi(depth)
			return v
		}
	}
	switch format {
	case "p010le", "p010be", "p210le", "p210be", "p410le", "p410be", "x2rgb10le", "x2bgr10le":
		return 10
	case "p012le", "p012be", "xyz12le", "xyz12be":
		return 12
	case "p016le", "p016be", "p216le", "p216be", "p416le", "p416be", "rgb48le", "rgb48be", "rgba64le", "rgba64be", "bgr48le", "bgr48be", "bgra64le", "bgra64be":
		return 16
	case "gbrpf32le", "gbrpf32be", "gbrapf32le", "gbrapf32be", "grayf32le", "grayf32be":
		return 32
	case "yuv420p", "yuv422p", "yuv444p", "yuvj420p", "yuvj422p", "yuvj444p", "yuv410p", "yuv411p", "yuva420p", "yuva422p", "yuva444p", "nv12", "nv21", "gray", "rgb24", "bgr24", "rgba", "argb", "bgra", "abgr", "gbrp", "pal8", "ya8":
		return 8
	default:
		return 0 // Unknown format depth is preferable to a fabricated 8-bit label.
	}
}
